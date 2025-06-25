package types

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// ResourceCollection holds all Gateway API Standard channel resources for v1.2.1 plus DNSRecord and Services
type ResourceCollection struct {
	GatewayClasses  []gatewayv1.GatewayClass        `json:"gatewayClasses"`
	Gateways        []gatewayv1.Gateway             `json:"gateways"`
	HTTPRoutes      []gatewayv1.HTTPRoute           `json:"httpRoutes"`
	ReferenceGrants []gatewayv1beta1.ReferenceGrant `json:"referenceGrants"`
	DNSRecords      []unstructured.Unstructured     `json:"dnsRecords"`
	Services        []corev1.Service                `json:"services"`
}

// Graph represents the graph structure for D3.js
type Graph struct {
	Nodes    []Node    `json:"nodes"`
	Links    []Link    `json:"links"`
	DNSZones []DNSZone `json:"dnsZones"`
}

// DNSZone represents a DNS zone grouping
type DNSZone struct {
	Name  string   `json:"name"`
	Nodes []string `json:"nodes"` // Node IDs that belong to this zone
	Color string   `json:"color"`
}

// Node represents a node in the graph
type Node struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Type         string        `json:"type"`
	Namespace    string        `json:"namespace"`
	Group        string        `json:"group"`
	Version      string        `json:"version"`
	Kind         string        `json:"kind"`
	ParentID     *string       `json:"parentId,omitempty"`     // For listener nodes, reference to parent Gateway
	ListenerData *ListenerData `json:"listenerData,omitempty"` // Additional data for listener nodes
	Hidden       bool          `json:"hidden,omitempty"`       // Whether node should be hidden by default
	DNSZone      string        `json:"dnsZone,omitempty"`      // DNS zone this resource belongs to
	Hostname     string        `json:"hostname,omitempty"`     // Hostname for DNSRecord and other hostname-based resources
}

// ListenerData contains additional information for Gateway listener nodes
type ListenerData struct {
	Port     int32   `json:"port"`
	Protocol string  `json:"protocol"`
	Hostname *string `json:"hostname,omitempty"`
	TLS      bool    `json:"tls"`
}

// Link represents a connection between nodes
type Link struct {
	Source int    `json:"source"`
	Target int    `json:"target"`
	Type   string `json:"type"`
}
