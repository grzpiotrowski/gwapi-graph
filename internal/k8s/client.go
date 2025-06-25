package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	corev1 "k8s.io/api/core/v1"
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

// GetServices returns all Service resources
func (c *Client) GetServices(ctx context.Context) ([]corev1.Service, error) {
	services, err := c.k8sClient.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list Services: %w", err)
	}

	return services.Items, nil
}

// GetGateway retrieves a specific Gateway resource
func (c *Client) GetGateway(ctx context.Context, namespace, name string) (*gatewayv1.Gateway, error) {
	gateway, err := c.gatewayClient.GatewayV1().Gateways(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway %s/%s: %w", namespace, name, err)
	}
	return gateway, nil
}

// GetHTTPRoute retrieves a specific HTTPRoute resource
func (c *Client) GetHTTPRoute(ctx context.Context, namespace, name string) (*gatewayv1.HTTPRoute, error) {
	route, err := c.gatewayClient.GatewayV1().HTTPRoutes(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTPRoute %s/%s: %w", namespace, name, err)
	}
	return route, nil
}

// GetGatewayClass retrieves a specific GatewayClass resource
func (c *Client) GetGatewayClass(ctx context.Context, name string) (*gatewayv1.GatewayClass, error) {
	class, err := c.gatewayClient.GatewayV1().GatewayClasses().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get GatewayClass %s: %w", name, err)
	}
	return class, nil
}

// GetReferenceGrant retrieves a specific ReferenceGrant resource
func (c *Client) GetReferenceGrant(ctx context.Context, namespace, name string) (*gatewayv1beta1.ReferenceGrant, error) {
	grant, err := c.gatewayClient.GatewayV1beta1().ReferenceGrants(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ReferenceGrant %s/%s: %w", namespace, name, err)
	}
	return grant, nil
}

// GetService retrieves a specific Service resource
func (c *Client) GetService(ctx context.Context, namespace, name string) (*corev1.Service, error) {
	service, err := c.k8sClient.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Service %s/%s: %w", namespace, name, err)
	}
	return service, nil
}

// GetDNSRecord retrieves a specific DNSRecord resource
func (c *Client) GetDNSRecord(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	gvr := schema.GroupVersionResource{
		Group:    "ingress.operator.openshift.io",
		Version:  "v1",
		Resource: "dnsrecords",
	}

	resource, err := c.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get DNSRecord %s/%s: %w", namespace, name, err)
	}
	return resource, nil
}

// UpdateGateway updates a Gateway resource
func (c *Client) UpdateGateway(ctx context.Context, namespace, name string, data map[string]interface{}) error {
	// Get the existing resource first
	existing, err := c.GetGateway(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Check for immutable field changes
	if metadata, ok := data["metadata"]; ok {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			if newName, exists := metadataMap["name"]; exists && newName != existing.Name {
				return fmt.Errorf("cannot change resource name from '%s' to '%s' - resource names are immutable", existing.Name, newName)
			}
			if newNamespace, exists := metadataMap["namespace"]; exists && newNamespace != existing.Namespace {
				return fmt.Errorf("cannot change resource namespace from '%s' to '%s' - resource namespaces are immutable", existing.Namespace, newNamespace)
			}
		}
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specBytes, &existing.Spec); err != nil {
			return fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	// Update mutable metadata fields (labels, annotations)
	if metadata, ok := data["metadata"]; ok {
		if metadataMap, ok := metadata.(map[string]interface{}); ok {
			if labels, exists := metadataMap["labels"]; exists {
				if labelsMap, ok := labels.(map[string]interface{}); ok {
					stringLabels := make(map[string]string)
					for k, v := range labelsMap {
						if str, ok := v.(string); ok {
							stringLabels[k] = str
						}
					}
					existing.Labels = stringLabels
				}
			}

			if annotations, exists := metadataMap["annotations"]; exists {
				if annotationsMap, ok := annotations.(map[string]interface{}); ok {
					stringAnnotations := make(map[string]string)
					for k, v := range annotationsMap {
						if str, ok := v.(string); ok {
							stringAnnotations[k] = str
						}
					}
					existing.Annotations = stringAnnotations
				}
			}
		}
	}

	_, err = c.gatewayClient.GatewayV1().Gateways(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Gateway %s/%s: %w", namespace, name, err)
	}
	return nil
}

// UpdateHTTPRoute updates an HTTPRoute resource
func (c *Client) UpdateHTTPRoute(ctx context.Context, namespace, name string, data map[string]interface{}) error {
	// Get the existing resource first
	existing, err := c.GetHTTPRoute(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specBytes, &existing.Spec); err != nil {
			return fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	_, err = c.gatewayClient.GatewayV1().HTTPRoutes(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update HTTPRoute %s/%s: %w", namespace, name, err)
	}
	return nil
}

// UpdateGatewayClass updates a GatewayClass resource
func (c *Client) UpdateGatewayClass(ctx context.Context, name string, data map[string]interface{}) error {
	// Get the existing resource first
	existing, err := c.GetGatewayClass(ctx, name)
	if err != nil {
		return err
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specBytes, &existing.Spec); err != nil {
			return fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	_, err = c.gatewayClient.GatewayV1().GatewayClasses().Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update GatewayClass %s: %w", name, err)
	}
	return nil
}

// UpdateReferenceGrant updates a ReferenceGrant resource
func (c *Client) UpdateReferenceGrant(ctx context.Context, namespace, name string, data map[string]interface{}) error {
	// Get the existing resource first
	existing, err := c.GetReferenceGrant(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specBytes, &existing.Spec); err != nil {
			return fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	_, err = c.gatewayClient.GatewayV1beta1().ReferenceGrants(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update ReferenceGrant %s/%s: %w", namespace, name, err)
	}
	return nil
}

// UpdateService updates a Service resource
func (c *Client) UpdateService(ctx context.Context, namespace, name string, data map[string]interface{}) error {
	// Get the existing resource first
	existing, err := c.GetService(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		specBytes, err := json.Marshal(spec)
		if err != nil {
			return fmt.Errorf("failed to marshal spec: %w", err)
		}
		if err := json.Unmarshal(specBytes, &existing.Spec); err != nil {
			return fmt.Errorf("failed to unmarshal spec: %w", err)
		}
	}

	_, err = c.k8sClient.CoreV1().Services(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update Service %s/%s: %w", namespace, name, err)
	}
	return nil
}

// UpdateDNSRecord updates a DNSRecord resource
func (c *Client) UpdateDNSRecord(ctx context.Context, namespace, name string, data map[string]interface{}) error {
	gvr := schema.GroupVersionResource{
		Group:    "ingress.operator.openshift.io",
		Version:  "v1",
		Resource: "dnsrecords",
	}

	// Get the existing resource first
	existing, err := c.GetDNSRecord(ctx, namespace, name)
	if err != nil {
		return err
	}

	// Update the spec if provided
	if spec, ok := data["spec"]; ok {
		if err := unstructured.SetNestedField(existing.Object, spec, "spec"); err != nil {
			return fmt.Errorf("failed to set spec: %w", err)
		}
	}

	_, err = c.dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, existing, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update DNSRecord %s/%s: %w", namespace, name, err)
	}
	return nil
}
