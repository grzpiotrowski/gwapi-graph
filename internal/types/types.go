package types

import (
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
)

// ResourceCollection holds all Gateway API Standard channel resources for v1.2.1
type ResourceCollection struct {
	GatewayClasses  []gatewayv1.GatewayClass        `json:"gatewayClasses"`
	Gateways        []gatewayv1.Gateway             `json:"gateways"`
	HTTPRoutes      []gatewayv1.HTTPRoute           `json:"httpRoutes"`
	ReferenceGrants []gatewayv1beta1.ReferenceGrant `json:"referenceGrants"`
}

// Graph represents the graph structure for D3.js
type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

// Node represents a node in the graph
type Node struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
	Group     string `json:"group"`
	Version   string `json:"version"`
	Kind      string `json:"kind"`
}

// Link represents a connection between nodes
type Link struct {
	Source int    `json:"source"`
	Target int    `json:"target"`
	Type   string `json:"type"`
}
