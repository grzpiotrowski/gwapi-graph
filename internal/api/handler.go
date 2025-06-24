package api

import (
	"context"
	"log"
	"net/http"
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

	log.Printf("Finished fetching resources. Total nodes that will be created: %d",
		len(collection.GatewayClasses)+len(collection.Gateways)+len(collection.HTTPRoutes)+len(collection.ReferenceGrants)+len(collection.DNSRecords))

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

	// Add DNSRecord nodes and links to Gateways
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

		// Link DNSRecord to Gateway based on the gateway label
		if labels, found, _ := unstructured.NestedMap(dns.Object, "metadata", "labels"); found {
			if gatewayName, exists := labels["gateway.networking.k8s.io/gateway-name"]; exists {
				gatewayNameStr, ok := gatewayName.(string)
				if ok {
					for _, gw := range resources.Gateways {
						if gw.Name == gatewayNameStr {
							link := types.Link{
								Source: nodeMap[string(gw.UID)],
								Target: nodeMap[node.ID],
								Type:   "dnsRecord",
							}
							graph.Links = append(graph.Links, link)
							break
						}
					}
				}
			}
		}
	}

	return graph
}
