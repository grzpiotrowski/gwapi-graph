package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"

	"gwapi-graph/internal/k8s"
	"gwapi-graph/internal/types"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for simplicity
	},
}

// Handler handles API requests
type Handler struct {
	k8sClient *k8s.Client
}

// NewHandler creates a new API handler
func NewHandler(k8sClient *k8s.Client) *Handler {
	return &Handler{
		k8sClient: k8sClient,
	}
}

// GetResources returns all Gateway API resources
func (h *Handler) GetResources(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resources, err := h.fetchAllResources(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resources)
}

// GetGraph returns the graph data structure for visualization
func (h *Handler) GetGraph(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resources, err := h.fetchAllResources(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	graph := h.buildGraph(resources)
	c.JSON(http.StatusOK, graph)
}

// HandleWebSocket handles WebSocket connections for real-time updates
func (h *Handler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}
	defer conn.Close()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			resources, err := h.fetchAllResources(ctx)
			cancel()

			if err != nil {
				log.Printf("Error fetching resources: %v", err)
				continue
			}

			graph := h.buildGraph(resources)
			if err := conn.WriteJSON(graph); err != nil {
				log.Printf("Error writing JSON: %v", err)
				return
			}
		}
	}
}

// fetchAllResources fetches all Gateway API Standard channel resources
func (h *Handler) fetchAllResources(ctx context.Context) (*types.ResourceCollection, error) {
	collection := &types.ResourceCollection{}

	log.Printf("Starting to fetch Gateway API resources...")

	// Fetch Gateway Classes
	gatewayClasses, err := h.k8sClient.GetGatewayClasses(ctx)
	if err != nil {
		log.Printf("Error fetching Gateway Classes: %v", err)
	} else {
		log.Printf("Found %d Gateway Classes", len(gatewayClasses))
		for _, gc := range gatewayClasses {
			log.Printf("  - GatewayClass: %s", gc.Name)
		}
		collection.GatewayClasses = gatewayClasses
	}

	// Fetch Gateways
	gateways, err := h.k8sClient.GetGateways(ctx)
	if err != nil {
		log.Printf("Error fetching Gateways: %v", err)
	} else {
		log.Printf("Found %d Gateways", len(gateways))
		for _, gw := range gateways {
			log.Printf("  - Gateway: %s/%s", gw.Namespace, gw.Name)
		}
		collection.Gateways = gateways
	}

	// Fetch HTTP Routes
	httpRoutes, err := h.k8sClient.GetHTTPRoutes(ctx)
	if err != nil {
		log.Printf("Error fetching HTTP Routes: %v", err)
	} else {
		log.Printf("Found %d HTTP Routes", len(httpRoutes))
		for _, route := range httpRoutes {
			log.Printf("  - HTTPRoute: %s/%s", route.Namespace, route.Name)
		}
		collection.HTTPRoutes = httpRoutes
	}

	// Fetch Reference Grants
	referenceGrants, err := h.k8sClient.GetReferenceGrants(ctx)
	if err != nil {
		log.Printf("Error fetching Reference Grants: %v", err)
	} else {
		log.Printf("Found %d Reference Grants", len(referenceGrants))
		for _, grant := range referenceGrants {
			log.Printf("  - ReferenceGrant: %s/%s", grant.Namespace, grant.Name)
		}
		collection.ReferenceGrants = referenceGrants
	}

	// Fetch DNSRecords
	dnsRecords, err := h.k8sClient.GetDNSRecords(ctx)
	if err != nil {
		log.Printf("Error fetching DNSRecords: %v", err)
	} else {
		log.Printf("Found %d DNSRecords", len(dnsRecords))
		for _, dns := range dnsRecords {
			name, _, _ := unstructured.NestedString(dns.Object, "metadata", "name")
			namespace, _, _ := unstructured.NestedString(dns.Object, "metadata", "namespace")
			log.Printf("  - DNSRecord: %s/%s", namespace, name)
		}
		collection.DNSRecords = dnsRecords
	}

	// Fetch Services
	services, err := h.k8sClient.GetServices(ctx)
	if err != nil {
		log.Printf("Error fetching Services: %v", err)
	} else {
		log.Printf("Found %d Services", len(services))
		for _, svc := range services {
			log.Printf("  - Service: %s/%s", svc.Namespace, svc.Name)
		}
		collection.Services = services
	}

	log.Printf("Finished fetching resources. Total nodes that will be created: %d",
		len(collection.GatewayClasses)+len(collection.Gateways)+len(collection.HTTPRoutes)+len(collection.ReferenceGrants)+len(collection.DNSRecords)+len(collection.Services))

	return collection, nil
}

// buildGraph creates a graph data structure from the resources
func (h *Handler) buildGraph(resources *types.ResourceCollection) *types.Graph {
	graph := &types.Graph{
		Nodes:    []types.Node{},
		Links:    []types.Link{},
		DNSZones: []types.DNSZone{},
	}

	nodeMap := make(map[string]int)
	nodeIndex := 0

	// Add GatewayClass nodes
	for _, gc := range resources.GatewayClasses {
		node := types.Node{
			ID:        string(gc.UID),
			Name:      gc.Name,
			Type:      "GatewayClass",
			Namespace: "", // GatewayClass is cluster-scoped
			Group:     "gateway.networking.k8s.io",
			Version:   "v1",
			Kind:      "GatewayClass",
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++
	}

	// Add Gateway nodes and links to GatewayClasses
	for _, gw := range resources.Gateways {
		node := types.Node{
			ID:        string(gw.UID),
			Name:      gw.Name,
			Type:      "Gateway",
			Namespace: gw.Namespace,
			Group:     "gateway.networking.k8s.io",
			Version:   "v1",
			Kind:      "Gateway",
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++

		// Add Gateway Listener nodes (hidden by default)
		for i, listener := range gw.Spec.Listeners {
			listenerID := fmt.Sprintf("%s-listener-%d", string(gw.UID), i)
			listenerName := string(listener.Name)
			if listenerName == "" {
				listenerName = fmt.Sprintf("listener-%d", i)
			}

			hostname := ""
			if listener.Hostname != nil {
				hostname = string(*listener.Hostname)
			}

			parentGatewayID := string(gw.UID)
			listenerNode := types.Node{
				ID:        listenerID,
				Name:      listenerName,
				Type:      "Listener",
				Namespace: gw.Namespace,
				Group:     "gateway.networking.k8s.io",
				Version:   "v1",
				Kind:      "Listener",
				ParentID:  &parentGatewayID,
				Hidden:    false, // Always visible
				ListenerData: &types.ListenerData{
					Port:     int32(listener.Port),
					Protocol: string(listener.Protocol),
					Hostname: func() *string {
						if hostname != "" {
							return &hostname
						}
						return nil
					}(),
					TLS: listener.TLS != nil,
				},
			}
			graph.Nodes = append(graph.Nodes, listenerNode)
			nodeMap[listenerNode.ID] = nodeIndex
			nodeIndex++

			// Link Listener to Gateway
			link := types.Link{
				Source: nodeMap[string(gw.UID)],
				Target: nodeMap[listenerID],
				Type:   "listener",
			}
			graph.Links = append(graph.Links, link)
		}

		// Link Gateway to GatewayClass
		if gw.Spec.GatewayClassName != "" {
			for _, gc := range resources.GatewayClasses {
				if string(gw.Spec.GatewayClassName) == gc.Name {
					link := types.Link{
						Source: nodeMap[string(gc.UID)],
						Target: nodeMap[node.ID],
						Type:   "gatewayClassRef",
					}
					graph.Links = append(graph.Links, link)
					break
				}
			}
		}
	}

	// Add HTTPRoute nodes and links to Gateways
	for _, route := range resources.HTTPRoutes {
		node := types.Node{
			ID:        string(route.UID),
			Name:      route.Name,
			Type:      "HTTPRoute",
			Namespace: route.Namespace,
			Group:     "gateway.networking.k8s.io",
			Version:   "v1",
			Kind:      "HTTPRoute",
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++

		// Link HTTPRoute to Gateways
		for _, parentRef := range route.Spec.ParentRefs {
			for _, gw := range resources.Gateways {
				if (parentRef.Name == "" || string(parentRef.Name) == gw.Name) &&
					(parentRef.Namespace == nil || string(*parentRef.Namespace) == route.Namespace || string(*parentRef.Namespace) == gw.Namespace) {
					link := types.Link{
						Source: nodeMap[string(gw.UID)],
						Target: nodeMap[node.ID],
						Type:   "parentRef",
					}
					graph.Links = append(graph.Links, link)
				}
			}
		}
	}

	// Add ReferenceGrant nodes
	for _, grant := range resources.ReferenceGrants {
		node := types.Node{
			ID:        string(grant.UID),
			Name:      grant.Name,
			Type:      "ReferenceGrant",
			Namespace: grant.Namespace,
			Group:     "gateway.networking.k8s.io",
			Version:   "v1beta1",
			Kind:      "ReferenceGrant",
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++
	}

	// Add DNSRecord nodes and links to Gateway Listeners
	for _, dns := range resources.DNSRecords {
		uid, _, _ := unstructured.NestedString(dns.Object, "metadata", "uid")
		name, _, _ := unstructured.NestedString(dns.Object, "metadata", "name")
		namespace, _, _ := unstructured.NestedString(dns.Object, "metadata", "namespace")

		// Get the DNS name from the DNSRecord spec
		dnsName, _, _ := unstructured.NestedString(dns.Object, "spec", "dnsName")
		// Remove trailing dot if present
		if strings.HasSuffix(dnsName, ".") {
			dnsName = strings.TrimSuffix(dnsName, ".")
		}

		node := types.Node{
			ID:        uid,
			Name:      name,
			Type:      "DNSRecord",
			Namespace: namespace,
			Group:     "ingress.operator.openshift.io",
			Version:   "v1",
			Kind:      "DNSRecord",
			Hostname:  dnsName,
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++

		// Link DNSRecord to specific Gateway Listener based on hostname matching
		if labels, found, _ := unstructured.NestedMap(dns.Object, "metadata", "labels"); found {
			if gatewayName, exists := labels["gateway.networking.k8s.io/gateway-name"]; exists {
				gatewayNameStr, ok := gatewayName.(string)
				if ok {
					// Get the DNS name from the DNSRecord spec
					dnsName, _, _ := unstructured.NestedString(dns.Object, "spec", "dnsName")
					// Remove trailing dot if present for comparison
					if strings.HasSuffix(dnsName, ".") {
						dnsName = strings.TrimSuffix(dnsName, ".")
					}

					// Find the matching Gateway and its listeners
					for _, gw := range resources.Gateways {
						if gw.Name == gatewayNameStr && gw.Namespace == namespace {
							// Try to match DNSRecord to specific listener by hostname
							linkedToListener := false
							for i, listener := range gw.Spec.Listeners {
								listenerID := fmt.Sprintf("%s-listener-%d", string(gw.UID), i)

								// Check if listener hostname matches the DNS name
								if listener.Hostname != nil && string(*listener.Hostname) == dnsName {
									if listenerIndex, exists := nodeMap[listenerID]; exists {
										link := types.Link{
											Source: listenerIndex,
											Target: nodeMap[node.ID],
											Type:   "dnsRecord",
										}
										graph.Links = append(graph.Links, link)
										linkedToListener = true
										break
									}
								}
							}

							// If no specific listener matched, fall back to linking to the Gateway itself
							// This handles wildcard DNSRecords or cases where hostname matching fails
							if !linkedToListener {
								if gatewayIndex, exists := nodeMap[string(gw.UID)]; exists {
									link := types.Link{
										Source: gatewayIndex,
										Target: nodeMap[node.ID],
										Type:   "dnsRecord",
									}
									graph.Links = append(graph.Links, link)
								}
							}
							break
						}
					}
				}
			}
		}
	}

	// Add Service nodes
	for _, svc := range resources.Services {
		node := types.Node{
			ID:        string(svc.UID),
			Name:      svc.Name,
			Type:      "Service",
			Namespace: svc.Namespace,
			Group:     "",
			Version:   "v1",
			Kind:      "Service",
		}
		graph.Nodes = append(graph.Nodes, node)
		nodeMap[node.ID] = nodeIndex
		nodeIndex++
	}

	// Extract DNS zones and assign them to nodes with hierarchical support
	dnsZoneMap := make(map[string][]string)    // zone name -> node IDs
	nodeZoneMap := make(map[string][]string)   // node ID -> all zones it belongs to
	nodePrimaryZone := make(map[string]string) // node ID -> primary (most specific) zone

	// Collect all DNSRecord hostnames to identify specific records
	dnsRecordHostnames := make(map[string]string) // hostname -> DNSRecord UID

	// First, collect all hostnames and their hierarchical zones from DNSRecords
	for _, dns := range resources.DNSRecords {
		dnsName, _, _ := unstructured.NestedString(dns.Object, "spec", "dnsName")
		dnsUID, _, _ := unstructured.NestedString(dns.Object, "metadata", "uid")

		// Remove trailing dot if present
		if strings.HasSuffix(dnsName, ".") {
			dnsName = strings.TrimSuffix(dnsName, ".")
		}

		if dnsName != "" {
			dnsRecordHostnames[dnsName] = dnsUID

			zones := h.extractHierarchicalZones(dnsName)
			if len(zones) > 0 {
				// Assign to all valid hierarchical zones
				for _, zone := range zones {
					dnsZoneMap[zone] = append(dnsZoneMap[zone], dnsUID)
					nodeZoneMap[dnsUID] = append(nodeZoneMap[dnsUID], zone)
				}
				// Set primary zone (most specific)
				if len(zones) > 0 {
					nodePrimaryZone[dnsUID] = zones[0]
					log.Printf("DNSRecord %s (%s) assigned to zones %v, primary: %s", dnsName, dnsUID, zones, zones[0])
				}
			}
		}
	}

	// Process HTTPRoutes and assign them to hierarchical zones based on their hostnames
	for _, route := range resources.HTTPRoutes {
		routeID := string(route.UID)

		// Check all hostnames in the HTTPRoute
		for _, hostname := range route.Spec.Hostnames {
			hostnameStr := string(hostname)

			// If this exact hostname has a DNSRecord, assign to all the same zones as the DNSRecord
			if dnsUID, hasSpecificDNSRecord := dnsRecordHostnames[hostnameStr]; hasSpecificDNSRecord {
				// Use the same zones as the DNSRecord
				if dnsZones, exists := nodeZoneMap[dnsUID]; exists {
					for _, zone := range dnsZones {
						dnsZoneMap[zone] = append(dnsZoneMap[zone], routeID)
						nodeZoneMap[routeID] = append(nodeZoneMap[routeID], zone)
					}
					if primaryZone, exists := nodePrimaryZone[dnsUID]; exists {
						nodePrimaryZone[routeID] = primaryZone
					}
					log.Printf("HTTPRoute %s/%s (%s) assigned to zones %v (matches DNSRecord)", route.Namespace, route.Name, routeID, dnsZones)
					break
				}
			}

			zones := h.extractHierarchicalZones(hostnameStr)
			if len(zones) > 0 {
				// Assign to all hierarchical zones
				for _, zone := range zones {
					dnsZoneMap[zone] = append(dnsZoneMap[zone], routeID)
					nodeZoneMap[routeID] = append(nodeZoneMap[routeID], zone)
				}
				// Set primary zone (most specific)
				if _, exists := nodePrimaryZone[routeID]; !exists {
					nodePrimaryZone[routeID] = zones[0]
				}
				log.Printf("HTTPRoute %s/%s (%s) assigned to zones %v, primary: %s", route.Namespace, route.Name, routeID, zones, zones[0])
				break // Only process first hostname for primary zone assignment
			}
		}
	}

	// Process Gateway listeners and assign them to hierarchical zones based on their hostnames
	for _, gw := range resources.Gateways {
		for i, listener := range gw.Spec.Listeners {
			listenerID := fmt.Sprintf("%s-listener-%d", string(gw.UID), i)

			if listener.Hostname != nil {
				hostnameStr := string(*listener.Hostname)

				// If this exact hostname has a DNSRecord, assign to all the same zones as the DNSRecord
				if dnsUID, hasSpecificDNSRecord := dnsRecordHostnames[hostnameStr]; hasSpecificDNSRecord {
					// Use the same zones as the DNSRecord
					if dnsZones, exists := nodeZoneMap[dnsUID]; exists {
						for _, zone := range dnsZones {
							dnsZoneMap[zone] = append(dnsZoneMap[zone], listenerID)
							nodeZoneMap[listenerID] = append(nodeZoneMap[listenerID], zone)
						}
						if primaryZone, exists := nodePrimaryZone[dnsUID]; exists {
							nodePrimaryZone[listenerID] = primaryZone
						}
						log.Printf("Gateway listener %s (%s) assigned to zones %v (matches DNSRecord)", gw.Name, listenerID, dnsZones)
						continue
					}
				}

				zones := h.extractHierarchicalZones(hostnameStr)
				if len(zones) > 0 {
					// Assign to all hierarchical zones
					for _, zone := range zones {
						dnsZoneMap[zone] = append(dnsZoneMap[zone], listenerID)
						nodeZoneMap[listenerID] = append(nodeZoneMap[listenerID], zone)
					}
					// Set primary zone (most specific)
					if _, exists := nodePrimaryZone[listenerID]; !exists {
						nodePrimaryZone[listenerID] = zones[0]
					}
					log.Printf("Gateway listener %s (%s) assigned to zones %v, primary: %s", gw.Name, listenerID, zones, zones[0])
				}
			}
		}
	}

	// Update nodes with their DNS zone information (use primary zone)
	for i := range graph.Nodes {
		if primaryZone, exists := nodePrimaryZone[graph.Nodes[i].ID]; exists {
			graph.Nodes[i].DNSZone = primaryZone
		}
	}

	// Create DNS zone objects with colors, but only for zones that provide meaningful separation
	zoneColors := []string{"#e3f2fd", "#f3e5f5", "#e8f5e8", "#fff3e0", "#fce4ec", "#e0f2f1", "#f9fbe7", "#fff8e1"}
	colorIndex := 0

	log.Printf("DNS Zone Summary:")

	// Sort zones by specificity (most specific first) to prioritize meaningful zones
	type zoneInfo struct {
		name    string
		nodeIDs []string
		depth   int
	}

	var zones []zoneInfo
	for zoneName, nodeIDs := range dnsZoneMap {
		zones = append(zones, zoneInfo{
			name:    zoneName,
			nodeIDs: nodeIDs,
			depth:   len(strings.Split(zoneName, ".")),
		})
		log.Printf("  Zone %s: %d nodes - %v", zoneName, len(nodeIDs), nodeIDs)
	}

	// Sort by depth (most specific first)
	sort.Slice(zones, func(i, j int) bool {
		return zones[i].depth > zones[j].depth
	})

	for _, zoneInfo := range zones {
		// Create all zones that have nodes, allowing hierarchical overlap
		// Only skip zones that would be identical to other zones (no meaningful separation)
		shouldCreateZone := true

		// Skip zones that are identical to a more specific zone
		for _, otherZone := range zones {
			if otherZone.depth > zoneInfo.depth && len(otherZone.nodeIDs) == len(zoneInfo.nodeIDs) {
				// Check if node sets are identical
				if h.slicesEqual(otherZone.nodeIDs, zoneInfo.nodeIDs) {
					shouldCreateZone = false
					break
				}
			}
		}

		if shouldCreateZone {
			zone := types.DNSZone{
				Name:  zoneInfo.name,
				Nodes: zoneInfo.nodeIDs,
				Color: zoneColors[colorIndex%len(zoneColors)],
			}
			graph.DNSZones = append(graph.DNSZones, zone)
			colorIndex++

			log.Printf("Created DNS zone %s with %d nodes: %v", zoneInfo.name, len(zoneInfo.nodeIDs), zoneInfo.nodeIDs)
		} else {
			log.Printf("Skipped DNS zone %s (identical to more specific zone)", zoneInfo.name)
		}
	}

	log.Printf("Total DNS zones created: %d", len(graph.DNSZones))

	// Link HTTPRoutes to Services via backendRefs
	for _, route := range resources.HTTPRoutes {
		for _, rule := range route.Spec.Rules {
			for _, backendRef := range rule.BackendRefs {
				// Find matching service
				for _, svc := range resources.Services {
					serviceName := string(backendRef.Name)
					serviceNamespace := route.Namespace // Default to route namespace
					if backendRef.Namespace != nil {
						serviceNamespace = string(*backendRef.Namespace)
					}

					if svc.Name == serviceName && svc.Namespace == serviceNamespace {
						link := types.Link{
							Source: nodeMap[string(route.UID)],
							Target: nodeMap[string(svc.UID)],
							Type:   "backendRef",
						}
						graph.Links = append(graph.Links, link)
						break
					}
				}
			}
		}
	}

	return graph
}

// hostnamesMatch checks if a DNS name matches a hostname pattern
// Supports exact matches and basic wildcard matching
func (h *Handler) hostnamesMatch(dnsName, routeHostname string) bool {
	// Exact match
	if dnsName == routeHostname {
		return true
	}

	// Wildcard matching - if route hostname starts with "*."
	if strings.HasPrefix(routeHostname, "*.") {
		wildcardDomain := strings.TrimPrefix(routeHostname, "*.")
		// Check if DNS name ends with the wildcard domain
		if strings.HasSuffix(dnsName, "."+wildcardDomain) || dnsName == wildcardDomain {
			return true
		}
	}

	// Check if DNS name matches subdomain pattern
	if strings.HasPrefix(dnsName, routeHostname+".") {
		return true
	}

	return false
}

// extractDNSZone extracts the DNS zone from a hostname with intelligent granularity
// Examples:
// - api.example.com -> example.com
// - *.gwapi.apps.ci-ln-xyz.gcp-2.ci.openshift.org -> gwapi.apps.ci-ln-xyz.gcp-2.ci.openshift.org
// - foo.abc.apps.ci-ln-xyz.gcp-2.ci.openshift.org -> abc.apps.ci-ln-xyz.gcp-2.ci.openshift.org
func (h *Handler) extractDNSZone(hostname string) string {
	if hostname == "" {
		return ""
	}

	// Remove wildcard prefix if present
	if strings.HasPrefix(hostname, "*.") {
		hostname = strings.TrimPrefix(hostname, "*.")
	}

	// Split hostname into parts
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return hostname // Single part, treat as zone itself
	}

	// Special handling for OpenShift/Kubernetes style domains
	// Pattern: [subdomain.]service.apps.cluster-name.domain.tld
	if h.isOpenShiftStyleDomain(parts) {
		return h.extractOpenShiftZone(parts)
	}

	// Special handling for internal cluster domains
	// Pattern: service.namespace.svc.cluster.local
	if h.isClusterInternalDomain(parts) {
		return h.extractClusterInternalZone(parts)
	}

	// For standard domains, use different granularity based on domain length
	if len(parts) >= 6 {
		// Very long domains - use last 4 parts for more granularity
		return strings.Join(parts[len(parts)-4:], ".")
	} else if len(parts) >= 4 {
		// Medium domains - use last 3 parts
		return strings.Join(parts[len(parts)-3:], ".")
	} else {
		// Short domains - use last 2 parts (standard)
		return strings.Join(parts[len(parts)-2:], ".")
	}
}

// isOpenShiftStyleDomain checks if this looks like an OpenShift cluster domain
// Pattern: *.apps.cluster-name.provider.region.domain.tld
func (h *Handler) isOpenShiftStyleDomain(parts []string) bool {
	if len(parts) < 6 {
		return false
	}

	// Look for common OpenShift patterns
	for i, part := range parts {
		if part == "apps" && i > 0 && i < len(parts)-3 {
			// Check if it looks like: something.apps.cluster.domain.tld
			return true
		}
	}

	return false
}

// extractOpenShiftZone extracts zone for OpenShift style domains
// *.gwapi.apps.cluster -> gwapi.apps.cluster...
// foo.abc.apps.cluster -> abc.apps.cluster...
func (h *Handler) extractOpenShiftZone(parts []string) string {
	// Find the "apps" part
	appsIndex := -1
	for i, part := range parts {
		if part == "apps" {
			appsIndex = i
			break
		}
	}

	if appsIndex == -1 {
		// Fallback to standard extraction
		return strings.Join(parts[len(parts)-3:], ".")
	}

	// Extract the service/application part before "apps"
	if appsIndex > 0 {
		// Include from the service level: service.apps.cluster.domain.tld
		return strings.Join(parts[appsIndex-1:], ".")
	} else {
		// apps is at the beginning, use everything
		return strings.Join(parts, ".")
	}
}

// isClusterInternalDomain checks for Kubernetes internal domains
// Pattern: service.namespace.svc.cluster.local
func (h *Handler) isClusterInternalDomain(parts []string) bool {
	if len(parts) < 3 {
		return false
	}

	// Look for cluster.local or svc.cluster.local patterns
	return (len(parts) >= 2 && parts[len(parts)-2] == "cluster" && parts[len(parts)-1] == "local") ||
		(len(parts) >= 4 && parts[len(parts)-4] == "svc" && parts[len(parts)-2] == "cluster" && parts[len(parts)-1] == "local")
}

// extractClusterInternalZone extracts zone for cluster internal domains
// service.namespace.svc.cluster.local -> namespace.svc.cluster.local
func (h *Handler) extractClusterInternalZone(parts []string) string {
	if len(parts) >= 4 && parts[len(parts)-4] == "svc" {
		// service.namespace.svc.cluster.local -> namespace.svc.cluster.local
		return strings.Join(parts[len(parts)-4:], ".")
	}

	// Fallback
	return strings.Join(parts[len(parts)-3:], ".")
}

// extractHierarchicalZones extracts all possible DNS zones from a hostname in hierarchical order
// Examples: foo.abc.apps.ci-ln-xyz.gcp-2.ci.openshift.org returns:
// - abc.apps.ci-ln-xyz.gcp-2.ci.openshift.org (most specific)
// - apps.ci-ln-xyz.gcp-2.ci.openshift.org
// - ci-ln-xyz.gcp-2.ci.openshift.org
// - gcp-2.ci.openshift.org (broader)
// - ci.openshift.org
// - openshift.org (broadest)
func (h *Handler) extractHierarchicalZones(hostname string) []string {
	if hostname == "" {
		return nil
	}

	// Remove wildcard prefix if present
	if strings.HasPrefix(hostname, "*.") {
		hostname = strings.TrimPrefix(hostname, "*.")
	}

	// Split hostname into parts
	parts := strings.Split(hostname, ".")
	if len(parts) < 2 {
		return []string{hostname}
	}

	var zones []string

	// For OpenShift style domains, create hierarchical zones
	if h.isOpenShiftStyleDomain(parts) {
		// Find the "apps" part
		appsIndex := -1
		for i, part := range parts {
			if part == "apps" {
				appsIndex = i
				break
			}
		}

		if appsIndex > 0 {
			// Start from the service level and work up
			// foo.abc.apps.cluster -> abc.apps.cluster, apps.cluster, cluster...
			for i := appsIndex - 1; i < len(parts)-1; i++ {
				if i >= 0 {
					zone := strings.Join(parts[i:], ".")
					zones = append(zones, zone)
				}
			}
		}
	} else {
		// For regular domains, create zones from specific to general
		// api.service.example.com -> service.example.com, example.com
		for i := len(parts) - 2; i >= 0; i-- {
			if i < len(parts)-1 { // Don't include the full hostname itself
				zone := strings.Join(parts[i:], ".")
				zones = append(zones, zone)
			}
		}
	}

	// Remove duplicates and ensure we have at least the basic zone
	uniqueZones := make(map[string]bool)
	var result []string

	for _, zone := range zones {
		if !uniqueZones[zone] && zone != "" {
			uniqueZones[zone] = true
			result = append(result, zone)
		}
	}

	// Ensure we have at least the basic zone extraction as fallback
	basicZone := h.extractDNSZone(hostname)
	if !uniqueZones[basicZone] && basicZone != "" {
		result = append(result, basicZone)
	}

	return result
}

// GetResourceDetails returns detailed information about a specific resource
func (h *Handler) GetResourceDetails(c *gin.Context) {
	resourceType := c.Param("type")
	resourceName := c.Param("name")
	namespace := c.Query("namespace")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resource interface{}
	var err error

	switch resourceType {
	case "gatewayclass":
		resource, err = h.k8sClient.GetGatewayClass(ctx, resourceName)
	case "gateway":
		resource, err = h.k8sClient.GetGateway(ctx, namespace, resourceName)
	case "httproute":
		resource, err = h.k8sClient.GetHTTPRoute(ctx, namespace, resourceName)
	case "referencegrant":
		resource, err = h.k8sClient.GetReferenceGrant(ctx, namespace, resourceName)
	case "service":
		resource, err = h.k8sClient.GetService(ctx, namespace, resourceName)
	case "dnsrecord":
		resource, err = h.k8sClient.GetDNSRecord(ctx, namespace, resourceName)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported resource type"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resource)
}

// UpdateResource updates a specific resource
func (h *Handler) UpdateResource(c *gin.Context) {
	resourceType := c.Param("type")
	resourceName := c.Param("name")
	namespace := c.Query("namespace")

	var rawResource map[string]interface{}
	if err := c.ShouldBindJSON(&rawResource); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var err error

	switch resourceType {
	case "gatewayclass":
		err = h.k8sClient.UpdateGatewayClass(ctx, resourceName, rawResource)
	case "gateway":
		err = h.k8sClient.UpdateGateway(ctx, namespace, resourceName, rawResource)
	case "httproute":
		err = h.k8sClient.UpdateHTTPRoute(ctx, namespace, resourceName, rawResource)
	case "referencegrant":
		err = h.k8sClient.UpdateReferenceGrant(ctx, namespace, resourceName, rawResource)
	case "service":
		err = h.k8sClient.UpdateService(ctx, namespace, resourceName, rawResource)
	case "dnsrecord":
		err = h.k8sClient.UpdateDNSRecord(ctx, namespace, resourceName, rawResource)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported resource type"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "resource updated successfully"})
}

// slicesEqual checks if two string slices contain the same elements (order doesn't matter)
func (h *Handler) slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	// Create maps to count occurrences
	countA := make(map[string]int)
	countB := make(map[string]int)

	for _, item := range a {
		countA[item]++
	}

	for _, item := range b {
		countB[item]++
	}

	// Compare maps
	for key, count := range countA {
		if countB[key] != count {
			return false
		}
	}

	return true
}
