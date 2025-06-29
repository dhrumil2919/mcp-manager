package kubernetes

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Client wraps the Kubernetes client
type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
}

// NewClient creates a new Kubernetes client
func NewClient(kubeconfig string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig == "" {
		// Try in-cluster config first
		config, err = rest.InClusterConfig()
		if err != nil {
			// Fall back to default kubeconfig location
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = filepath.Join(home, ".kube", "config")
			}
		}
	}

	if config == nil {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// MCPServerInfo represents information about an MCP server
type MCPServerInfo struct {
	Name          string
	Image         string
	Status        string
	Replicas      int32
	ReadyReplicas int32
	Age           string
	Endpoint      string
	Namespace     string
}

// ListMCPServers lists all MCP servers in a namespace
func (c *Client) ListMCPServers(namespace string) ([]*MCPServerInfo, error) {
	deployments, err := c.clientset.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=mcp-manager",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var servers []*MCPServerInfo
	for _, deployment := range deployments.Items {
		server := &MCPServerInfo{
			Name:          deployment.Name,
			Namespace:     deployment.Namespace,
			Replicas:      *deployment.Spec.Replicas,
			ReadyReplicas: deployment.Status.ReadyReplicas,
			Age:           time.Since(deployment.CreationTimestamp.Time).Truncate(time.Second).String(),
		}

		// Get image from first container
		if len(deployment.Spec.Template.Spec.Containers) > 0 {
			server.Image = deployment.Spec.Template.Spec.Containers[0].Image
		}

		// Determine status
		if deployment.Status.ReadyReplicas == *deployment.Spec.Replicas {
			server.Status = "Running"
		} else if deployment.Status.ReadyReplicas > 0 {
			server.Status = "Partial"
		} else {
			server.Status = "Pending"
		}

		// Get endpoint from ingress
		endpoint, err := c.getServerEndpoint(deployment.Name, namespace)
		if err == nil {
			server.Endpoint = endpoint
		} else {
			server.Endpoint = "No ingress"
		}

		servers = append(servers, server)
	}

	return servers, nil
}

// getServerEndpoint gets the ingress endpoint for a server
func (c *Client) getServerEndpoint(serverName, namespace string) (string, error) {
	ingresses, err := c.clientset.NetworkingV1().Ingresses(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", serverName),
	})
	if err != nil {
		return "", err
	}

	if len(ingresses.Items) == 0 {
		return "", fmt.Errorf("no ingress found")
	}

	ingress := ingresses.Items[0]
	if len(ingress.Spec.Rules) > 0 {
		rule := ingress.Spec.Rules[0]
		protocol := "http"
		if ingress.Spec.TLS != nil && len(ingress.Spec.TLS) > 0 {
			protocol = "https"
		}
		return fmt.Sprintf("%s://%s", protocol, rule.Host), nil
	}

	return "", fmt.Errorf("no rules found in ingress")
}

// DeploymentConfig represents deployment configuration
type DeploymentConfig struct {
	Name          string
	Image         string
	Port          int
	HealthPath    string
	Command       []string
	Args          []string
	Environment   map[string]string
	SecretsFile   string
	Namespace     string
	Replicas      int
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	IngressHost   string
	IngressClass  string
	Labels        map[string]string
}

// DeployMCPServer deploys an MCP server to Kubernetes
func (c *Client) DeployMCPServer(config *DeploymentConfig) error {
	// Create deployment
	if err := c.createDeployment(config); err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Create service
	if err := c.createService(config); err != nil {
		return fmt.Errorf("failed to create service: %w", err)
	}

	// Create ingress if host is specified
	if config.IngressHost != "" {
		if err := c.createIngress(config); err != nil {
			return fmt.Errorf("failed to create ingress: %w", err)
		}
	}

	return nil
}

// createDeployment creates a Kubernetes deployment
func (c *Client) createDeployment(config *DeploymentConfig) error {
	labels := map[string]string{
		"app.kubernetes.io/name":       config.Name,
		"app.kubernetes.io/managed-by": "mcp-manager",
		"app.kubernetes.io/component":  "mcp-server",
	}

	// Add custom labels
	for k, v := range config.Labels {
		labels[k] = v
	}

	// Build environment variables
	var envVars []corev1.EnvVar
	for k, v := range config.Environment {
		envVars = append(envVars, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// Build resource requirements
	resources := corev1.ResourceRequirements{}
	if config.CPURequest != "" || config.MemoryRequest != "" {
		resources.Requests = corev1.ResourceList{}
		if config.CPURequest != "" {
			resources.Requests[corev1.ResourceCPU] = parseQuantity(config.CPURequest)
		}
		if config.MemoryRequest != "" {
			resources.Requests[corev1.ResourceMemory] = parseQuantity(config.MemoryRequest)
		}
	}
	if config.CPULimit != "" || config.MemoryLimit != "" {
		resources.Limits = corev1.ResourceList{}
		if config.CPULimit != "" {
			resources.Limits[corev1.ResourceCPU] = parseQuantity(config.CPULimit)
		}
		if config.MemoryLimit != "" {
			resources.Limits[corev1.ResourceMemory] = parseQuantity(config.MemoryLimit)
		}
	}

	// Build container
	container := corev1.Container{
		Name:      config.Name,
		Image:     config.Image,
		Ports:     []corev1.ContainerPort{{ContainerPort: int32(config.Port)}},
		Env:       envVars,
		Resources: resources,
	}

	if len(config.Command) > 0 {
		container.Command = config.Command
	}
	if len(config.Args) > 0 {
		container.Args = config.Args
	}

	// Add health check if specified
	if config.HealthPath != "" {
		container.LivenessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: config.HealthPath,
					Port: intstr.FromInt(config.Port),
				},
			},
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
		}
		container.ReadinessProbe = &corev1.Probe{
			ProbeHandler: corev1.ProbeHandler{
				HTTPGet: &corev1.HTTPGetAction{
					Path: config.HealthPath,
					Port: intstr.FromInt(config.Port),
				},
			},
			InitialDelaySeconds: 5,
			PeriodSeconds:       5,
		}
	}

	replicas := int32(config.Replicas)
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app.kubernetes.io/name": config.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{container},
				},
			},
		},
	}

	_, err := c.clientset.AppsV1().Deployments(config.Namespace).Create(context.TODO(), deployment, metav1.CreateOptions{})
	return err
}

// Helper function to parse resource quantities
func parseQuantity(s string) resource.Quantity {
	q, _ := resource.ParseQuantity(s)
	return q
}

// createService creates a Kubernetes service
func (c *Client) createService(config *DeploymentConfig) error {
	labels := map[string]string{
		"app.kubernetes.io/name":       config.Name,
		"app.kubernetes.io/managed-by": "mcp-manager",
		"app.kubernetes.io/component":  "mcp-server",
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app.kubernetes.io/name": config.Name,
			},
			Ports: []corev1.ServicePort{
				{
					Port:       int32(config.Port),
					TargetPort: intstr.FromInt(config.Port),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}

	_, err := c.clientset.CoreV1().Services(config.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	return err
}

// createIngress creates a Kubernetes ingress
func (c *Client) createIngress(config *DeploymentConfig) error {
	labels := map[string]string{
		"app.kubernetes.io/name":       config.Name,
		"app.kubernetes.io/managed-by": "mcp-manager",
		"app.kubernetes.io/component":  "mcp-server",
	}

	pathType := networkingv1.PathTypePrefix
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      config.Name,
			Namespace: config.Namespace,
			Labels:    labels,
			Annotations: map[string]string{
				"kubernetes.io/ingress.class": config.IngressClass,
			},
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: config.IngressHost,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &pathType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: config.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(config.Port),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	_, err := c.clientset.NetworkingV1().Ingresses(config.Namespace).Create(context.TODO(), ingress, metav1.CreateOptions{})
	return err
}

// HealthResult represents health check result
type HealthResult struct {
	Status       string        `json:"status"`
	Endpoint     string        `json:"endpoint"`
	ResponseTime time.Duration `json:"response_time"`
	Message      string        `json:"message"`
	Details      interface{}   `json:"details,omitempty"`
}

// CheckServerHealth checks the health of a server
func (c *Client) CheckServerHealth(serverName, namespace, endpoint string, timeout time.Duration) (*HealthResult, error) {
	// Implementation would make HTTP request to the server's health endpoint
	// For now, return a basic implementation
	return &HealthResult{
		Status:   "healthy",
		Endpoint: fmt.Sprintf("http://%s.%s.svc.cluster.local%s", serverName, namespace, endpoint),
		Message:  "Health check not implemented yet",
	}, nil
}

// LogOptions represents log fetching options
type LogOptions struct {
	Follow     bool
	Tail       int
	Since      string
	Timestamps bool
	Container  string
	Previous   bool
}

// GetServerLogs gets logs from a server
func (c *Client) GetServerLogs(serverName, namespace string, options *LogOptions) error {
	// Get pods for the deployment
	pods, err := c.clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", serverName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for server %s", serverName)
	}

	// Use the first pod
	pod := pods.Items[0]

	// Build log options
	logOptions := &corev1.PodLogOptions{
		Follow:     options.Follow,
		Timestamps: options.Timestamps,
		Previous:   options.Previous,
	}

	if options.Tail > 0 {
		tail := int64(options.Tail)
		logOptions.TailLines = &tail
	}

	if options.Since != "" {
		// Parse since time
		if sinceTime, err := time.Parse(time.RFC3339, options.Since); err == nil {
			metaTime := metav1.NewTime(sinceTime)
			logOptions.SinceTime = &metaTime
		}
	}

	if options.Container != "" {
		logOptions.Container = options.Container
	}

	// Get log stream
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, logOptions)
	stream, err := req.Stream(context.TODO())
	if err != nil {
		return fmt.Errorf("failed to get log stream: %w", err)
	}
	defer stream.Close()

	// Copy logs to stdout
	_, err = io.Copy(os.Stdout, stream)
	return err
}

// DeleteOptions represents deletion options
type DeleteOptions struct {
	Namespace   string
	KeepVolumes bool
}

// ServerExists checks if a server exists
func (c *Client) ServerExists(serverName, namespace string) (bool, error) {
	_, err := c.clientset.AppsV1().Deployments(namespace).Get(context.TODO(), serverName, metav1.GetOptions{})
	if err != nil {
		return false, nil
	}
	return true, nil
}

// DeleteMCPServer deletes an MCP server and its resources
func (c *Client) DeleteMCPServer(serverName string, options *DeleteOptions) error {
	// Delete deployment
	err := c.clientset.AppsV1().Deployments(options.Namespace).Delete(context.TODO(), serverName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}

	// Delete service
	err = c.clientset.CoreV1().Services(options.Namespace).Delete(context.TODO(), serverName, metav1.DeleteOptions{})
	if err != nil {
		// Service might not exist, log but don't fail
		fmt.Printf("Warning: failed to delete service: %v\n", err)
	}

	// Delete ingress
	err = c.clientset.NetworkingV1().Ingresses(options.Namespace).Delete(context.TODO(), serverName, metav1.DeleteOptions{})
	if err != nil {
		// Ingress might not exist, log but don't fail
		fmt.Printf("Warning: failed to delete ingress: %v\n", err)
	}

	return nil
}
