package k8s

import (
	"context"
	"fmt"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// Client wraps Kubernetes and Gateway API clients
type Client struct {
	k8sClient     kubernetes.Interface
	gatewayClient gatewayclient.Interface
	dynamicClient dynamic.Interface
}

// NewClient creates a new Kubernetes client
func NewClient() (*Client, error) {
	config, err := getConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	gatewayClient, err := gatewayclient.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gateway API client: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return &Client{
		k8sClient:     k8sClient,
		gatewayClient: gatewayClient,
		dynamicClient: dynamicClient,
	}, nil
}

// getConfig returns the Kubernetes configuration
func getConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from flags: %w", err)
	}

	return config, nil
}

// GetGateways retrieves all Gateway resources
func (c *Client) GetGateways(ctx context.Context) ([]gatewayv1.Gateway, error) {
	gateways, err := c.gatewayClient.GatewayV1().Gateways("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list gateways: %w", err)
	}
	return gateways.Items, nil
}

// GetHTTPRoutes retrieves all HTTPRoute resources
func (c *Client) GetHTTPRoutes(ctx context.Context) ([]gatewayv1.HTTPRoute, error) {
	routes, err := c.gatewayClient.GatewayV1().HTTPRoutes("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list HTTP routes: %w", err)
	}
	return routes.Items, nil
}

// GetGatewayClasses retrieves all GatewayClass resources
func (c *Client) GetGatewayClasses(ctx context.Context) ([]gatewayv1.GatewayClass, error) {
	classes, err := c.gatewayClient.GatewayV1().GatewayClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list gateway classes: %w", err)
	}
	return classes.Items, nil
}

// GetReferenceGrants retrieves all ReferenceGrant resources (v1beta1 in Gateway API v1.2.1)
func (c *Client) GetReferenceGrants(ctx context.Context) ([]gatewayv1beta1.ReferenceGrant, error) {
	grants, err := c.gatewayClient.GatewayV1beta1().ReferenceGrants("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list reference grants: %w", err)
	}
	return grants.Items, nil
}

// GetDNSRecords returns all DNSRecord resources
func (c *Client) GetDNSRecords(ctx context.Context) ([]unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    "ingress.operator.openshift.io",
		Version:  "v1",
		Resource: "dnsrecords",
	}

	result, err := c.dynamicClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list DNSRecords: %w", err)
	}

	return result.Items, nil
}
