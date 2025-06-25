package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
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
		Nodes: []types.Node{},
		Links: []types.Link{},
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

		node := types.Node{
			ID:        uid,
			Name:      name,
			Type:      "DNSRecord",
			Namespace: namespace,
			Group:     "ingress.operator.openshift.io",
			Version:   "v1",
			Kind:      "DNSRecord",
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
