//go:build ignore

package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// KubernetesClient provides integration with Kubernetes API for service discovery
type KubernetesClient struct {
	mu          sync.RWMutex
	clientset   *kubernetes.Clientset
	dynamicClient *dynamic.DynamicClient
	config      *DiscoveryConfig
	logger      *logrus.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	restCfg     *rest.Config
	initialized bool
	watchers    map[string]cache.Controller
	eventHandlers map[EventType][]func(*Event)
}

// NewKubernetesClient creates a new Kubernetes client instance
func NewKubernetesClient(cfg *DiscoveryConfig, logger *logrus.Logger) *KubernetesClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &KubernetesClient{
		config:      cfg,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		watchers:    make(map[string]cache.Controller),
		eventHandlers: make(map[EventType][]func(*Event)),
		initialized: false,
	}
}

// SetConfig updates the configuration
func (kc *KubernetesClient) SetConfig(cfg *DiscoveryConfig) {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	kc.config = cfg
}

// GetConfig returns the current configuration
func (kc *KubernetesClient) GetConfig() *DiscoveryConfig {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.config
}

// initializeClient initializes the Kubernetes client connection
func (kc *KubernetesClient) initializeClient() error {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.initialized && kc.clientset != nil {
		return nil
	}

	var err error

	// Try in-cluster config first (for running inside a pod)
	kc.restCfg, err = rest.InClusterConfig()
	if err == nil {
		kc.clientset, err = kubernetes.NewForConfig(kc.restCfg)
		if err == nil {
			kc.dynamicClient, err = dynamic.NewForConfig(kc.restCfg)
			if err == nil {
				kc.logger.Info("Connected to Kubernetes cluster (in-cluster)")
				kc.initialized = true
				return nil
			}
		}
	}

	// Fall back to kubeconfig file
	kubeconfigPath := kc.config.KubeconfigPath
	if kubeconfigPath == "" {
		// Default to $HOME/.kube/config
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfigPath != "" {
		kc.restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err == nil {
			kc.clientset, err = kubernetes.NewForConfig(kc.restCfg)
			if err == nil {
				kc.dynamicClient, err = dynamic.NewForConfig(kc.restCfg)
				if err == nil {
					kc.logger.WithFields(logrus.Fields{
						"config": kubeconfigPath,
					}).Info("Connected to Kubernetes cluster (kubeconfig)")
					kc.initialized = true
					return nil
				}
			}
		}
	}

	// Try default config location
	kc.restCfg, err = clientcmd.LoadFromFile(clientcmd.RecommendedHomeFile)
	if err == nil {
		kc.clientset, err = kubernetes.NewForConfig(kc.restCfg)
		if err == nil {
			kc.dynamicClient, err = dynamic.NewForConfig(kc.restCfg)
			if err == nil {
				kc.logger.Info("Connected to Kubernetes cluster (default config)")
				kc.initialized = true
				return nil
			}
		}
	}

	// Error out with helpful message
	if err != nil {
		kc.logger.WithFields(logrus.Fields{
			"error":      err,
			"config":     kubeconfigPath,
		}).Error("Failed to initialize Kubernetes client")
		return fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}

	return nil
}

// Ping checks the Kubernetes cluster connection
func (kc *KubernetesClient) Ping() error {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return ErrK8sNotInitialized
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 10*time.Second)
	defer cancel()

	_, err := clientset.Discovery().ServerVersion()
	return err
}

// IsConnected returns true if the client is connected to Kubernetes
func (kc *KubernetesClient) IsConnected() bool {
	return kc.clientset != nil && kc.initialized
}

// GetConfigMap retrieves a ConfigMap
func (kc *KubernetesClient) GetConfigMap(namespace, name string) (*corev1.ConfigMap, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	configMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get configmap %s/%s: %w", namespace, name, err)
	}

	return configMap, nil
}

// ListConfigMaps lists ConfigMaps in a namespace
func (kc *KubernetesClient) ListConfigMaps(namespace string, labels string) ([]*corev1.ConfigMap, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps in %s: %w", namespace, err)
	}

	result := make([]*corev1.ConfigMap, len(configMaps.Items))
	for i := range configMaps.Items {
		result[i] = &configMaps.Items[i]
	}

	return result, nil
}

// GetSecret retrieves a Secret
func (kc *KubernetesClient) GetSecret(namespace, name string) (*corev1.Secret, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s/%s: %w", namespace, name, err)
	}

	return secret, nil
}

// ListSecrets lists Secrets in a namespace
func (kc *KubernetesClient) ListSecrets(namespace string, labels string) ([]*corev1.Secret, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets in %s: %w", namespace, err)
	}

	result := make([]*corev1.Secret, len(secrets.Items))
	for i := range secrets.Items {
		result[i] = &secrets.Items[i]
	}

	return result, nil
}

// GetService retrieves a Service
func (kc *KubernetesClient) GetService(namespace, name string) (*corev1.Service, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	service, err := clientset.CoreV1().Services(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get service %s/%s: %w", namespace, name, err)
	}

	return service, nil
}

// ListServices lists Services in a namespace
func (kc *KubernetesClient) ListServices(namespace string, labels string) ([]*corev1.Service, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	services, err := clientset.CoreV1().Services(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list services in %s: %w", namespace, err)
	}

	result := make([]*corev1.Service, len(services.Items))
	for i := range services.Items {
		result[i] = &services.Items[i]
	}

	return result, nil
}

// GetPod retrieves a Pod
func (kc *KubernetesClient) GetPod(namespace, name string) (*corev1.Pod, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s/%s: %w", namespace, name, err)
	}

	return pod, nil
}

// ListPods lists Pods in a namespace
func (kc *KubernetesClient) ListPods(namespace string, labels string) ([]*corev1.Pod, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pods in %s: %w", namespace, err)
	}

	result := make([]*corev1.Pod, len(pods.Items))
	for i := range pods.Items {
		result[i] = &pods.Items[i]
	}

	return result, nil
}

// GetDeployment retrieves a Deployment
func (kc *KubernetesClient) GetDeployment(namespace, name string) (*appsv1.Deployment, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment %s/%s: %w", namespace, name, err)
	}

	return deployment, nil
}

// ListDeployments lists Deployments in a namespace
func (kc *KubernetesClient) ListDeployments(namespace string, labels string) ([]*appsv1.Deployment, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments in %s: %w", namespace, err)
	}

	result := make([]*appsv1.Deployment, len(deployments.Items))
	for i := range deployments.Items {
		result[i] = &deployments.Items[i]
	}

	return result, nil
}

// GetStatefulSet retrieves a StatefulSet
func (kc *KubernetesClient) GetStatefulSet(namespace, name string) (*appsv1.StatefulSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	statefulset, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get statefulset %s/%s: %w", namespace, name, err)
	}

	return statefulset, nil
}

// ListStatefulSets lists StatefulSets in a namespace
func (kc *KubernetesClient) ListStatefulSets(namespace string, labels string) ([]*appsv1.StatefulSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list statefulsets in %s: %w", namespace, err)
	}

	result := make([]*appsv1.StatefulSet, len(statefulsets.Items))
	for i := range statefulsets.Items {
		result[i] = &statefulsets.Items[i]
	}

	return result, nil
}

// GetDaemonSet retrieves a DaemonSet
func (kc *KubernetesClient) GetDaemonSet(namespace, name string) (*appsv1.DaemonSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	daemonset, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get daemonset %s/%s: %w", namespace, name, err)
	}

	return daemonset.GroupVersionKind(), nil
}

// ListDaemonSets lists DaemonSets in a namespace
func (kc *KubernetesClient) ListDaemonSets(namespace string, labels string) ([]*appsv1.DaemonSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list daemonsets in %s: %w", namespace, err)
	}

	result := make([]*runtime.GroupVersionKind, len(daemonsets.Items))
	for i := range daemonsets.Items {
		result[i] = &daemonsets.Items[i].GroupVersionKind()
	}

	return result, nil
}

// GetReplicaSet retrieves a ReplicaSet
func (kc *KubernetesClient) GetReplicaSet(namespace, name string) (*appsv1.ReplicaSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	replicaset, err := clientset.AppsV1().ReplicaSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get replicaset %s/%s: %w", namespace, name, err)
	}

	return replicaset.GroupVersionKind(), nil
}

// ListReplicaSets lists ReplicaSets in a namespace
func (kc *KubernetesClient) ListReplicaSets(namespace string, labels string) ([]*appsv1.ReplicaSet, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list replica sets in %s: %w", namespace, err)
	}

	result := make([]*runtime.GroupVersionKind, len(replicaSets.Items))
	for i := range replicaSets.Items {
		result[i] = &replicaSets.Items[i].GroupVersionKind()
	}

	return result, nil
}

// GetIngress retrieves an Ingress
func (kc *KubernetesClient) GetIngress(namespace, name string) (*networkingv1.Ingress, error) {
	kc.mu.RLock()
	dynamicClient := kc.dynamicClient
	kc.mu.RUnlock()

	if dynamicClient == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	// Use dynamic client for Ingress
	ingressClient := dynamicClient.Resource(networkingv1.SchemeGroupVersion.WithResource("ingresses")).Namespace(namespace)

	obj, err := ingressClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get ingress %s/%s: %w", namespace, name, err)
	}

	// Convert to Ingress type
	ingress := &networkingv1.Ingress{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.UnstructuredContent(), ingress); err != nil {
		return nil, fmt.Errorf("failed to convert ingress: %w", err)
	}

	return ingress, nil
}

// ListIngresses lists Ingresses in a namespace
func (kc *KubernetesClient) ListIngresses(namespace string) ([]*networkingv1.Ingress, error) {
	kc.mu.RLock()
	dynamicClient := kc.dynamicClient
	kc.mu.RUnlock()

	if dynamicClient == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	// Use dynamic client for Ingress
	ingressClient := dynamicClient.Resource(networkingv1.SchemeGroupVersion.WithResource("ingresses")).Namespace(namespace)

	ingresses, err := ingressClient.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list ingresses in %s: %w", namespace, err)
	}

	result := make([]*networkingv1.Ingress, len(ingresses.Items))
	for i := range ingresses.Items {
		item := ingresses.Items[i]
		ingress := &networkingv1.Ingress{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), ingress); err != nil {
			kc.logger.WithError(err).Warnf("Failed to convert ingress item: %v", item)
			continue
		}
		result[i] = ingress
	}

	return result, nil
}

// GetPersistentVolumeClaim retrieves a PVC
func (kc *KubernetesClient) GetPersistentVolumeClaim(namespace, name string) (*corev1.PersistentVolumeClaim, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pvc %s/%s: %w", namespace, name, err)
	}

	return pvc, nil
}

// ListPersistentVolumeClaims lists PVCs in a namespace
func (kc *KubernetesClient) ListPersistentVolumeClaims(namespace string, labels string) ([]*corev1.PersistentVolumeClaim, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list pvc in %s: %w", namespace, err)
	}

	result := make([]*corev1.PersistentVolumeClaim, len(pvcs.Items))
	for i := range pvcs.Items {
		result[i] = &pvcs.Items[i]
	}

	return result, nil
}

// GetNode retrieves a Node
func (kc *KubernetesClient) GetNode(name string) (*corev1.Node, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	node, err := clientset.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node %s: %w", name, err)
	}

	return node, nil
}

// ListNodes lists all Nodes
func (kc *KubernetesClient) ListNodes(labels string) ([]*corev1.Node, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	result := make([]*corev1.Node, len(nodes.Items))
	for i := range nodes.Items {
		result[i] = &nodes.Items[i]
	}

	return result, nil
}

// GetNamespace retrieves a Namespace
func (kc *KubernetesClient) GetNamespace(name string) (*corev1.Namespace, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	return namespace, nil
}

// ListNamespaces lists all Namespaces
func (kc *KubernetesClient) ListNamespaces(labels string) ([]*corev1.Namespace, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 30*time.Second)
	defer cancel()

	opts := metav1.ListOptions{}
	if labels != "" {
		opts.LabelSelector = labels
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	result := make([]*corev1.Namespace, len(namespaces.Items))
	for i := range namespaces.Items {
		result[i] = &namespaces.Items[i]
	}

	return result, nil
}

// ExecuteCommand executes a command in a container
func (kc *KubernetesClient) ExecuteCommand(namespace, podName, containerName string, command []string) (string, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return "", ErrK8sNotConnected
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 60*time.Second)
	defer cancel()

	// TODO: Implement command execution
	// For now, return an error
	return "", fmt.Errorf("command execution not implemented")
}

// GetLogs retrieves logs from a container
func (kc *KubernetesClient) GetLogs(namespace, podName, containerName string, since time.Time, follow bool) (<-chan string, <-chan error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 60*time.Second)
	defer cancel()

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &v1.PodLogOptions{
		Container: containerName,
		SinceTime: &metav1.Time{Time: since},
		Follow:    follow,
	})

	logChan := make(chan string, 100)
	errChan := make(chan error, 1)

	go func() {
		defer close(logChan)
		defer close(errChan)

		stream, err := req.Stream(ctx)
		if err != nil {
			errChan <- err
			return
		}
		defer stream.Close()

		scanner := bufio.NewScanner(stream)
		for scanner.Scan() {
			logChan <- scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			errChan <- err
		}
	}()

	return logChan, errChan
}

// WatchEvents watches Kubernetes events for changes
func (kc *KubernetesClient) WatchEvents(handler func(*Event)) {
	go func() {
		eventCtx, cancel := context.WithCancel(kc.ctx)
		defer cancel()

		for {
			select {
			case <-eventCtx.Done():
				return
			default:
				// Watch for pod events
				go kc.watchPods(eventCtx, handler)
				go kc.watchDeployments(eventCtx, handler)
				go kc.watchServices(eventCtx, handler)

				time.Sleep(10 * time.Second)
			}
		}
	}()
}

// watchPods watches for pod changes
func (kc *KubernetesClient) watchPods(ctx context.Context, handler func(*Event)) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return
	}

	podsClient := clientset.CoreV1().Pods(v1.NamespaceAll)

	watcher, err := podsClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		kc.logger.WithError(err).Error("Failed to start pod watcher")
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			kc.handlePodEvent(event.Type, pod, handler)
		}
	}
}

// watchDeployments watches for deployment changes
func (kc *KubernetesClient) watchDeployments(ctx context.Context, handler func(*Event)) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return
	}

	deploymentsClient := clientset.AppsV1().Deployments(v1.NamespaceAll)

	watcher, err := deploymentsClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		kc.logger.WithError(err).Error("Failed to start deployment watcher")
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			deployment, ok := event.Object.(*corev1.Deployment)
			if !ok {
				continue
			}

			kc.handleDeploymentEvent(event.Type, deployment, handler)
		}
	}
}

// watchServices watches for service changes
func (kc *KubernetesClient) watchServices(ctx context.Context, handler func(*Event)) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return
	}

	servicesClient := clientset.CoreV1().Services(v1.NamespaceAll)

	watcher, err := servicesClient.Watch(ctx, metav1.ListOptions{})
	if err != nil {
		kc.logger.WithError(err).Error("Failed to start service watcher")
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return
			}

			service, ok := event.Object.(*corev1.Service)
			if !ok {
				continue
			}

			kc.handleServiceEvent(event.Type, service, handler)
		}
	}
}

// handlePodEvent handles pod change events
func (kc *KubernetesClient) handlePodEvent(eventType watch.EventType, pod *corev1.Pod, handler func(*Event)) {
	event := &Event{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Map watch events to our event types
	switch eventType {
	case watch.Added:
		event.Type = EventPodAdded
		event.Pod = &PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
		}
	case watch.Modified:
		event.Type = EventPodUpdated
		event.Pod = &PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    string(pod.Status.Phase),
		}
		event.Metadata["old_status"] = pod.Status.Phase
	case watch.Deleted:
		event.Type = EventPodDeleted
		event.Pod = &PodDetail{
			Name:      pod.Name,
			Namespace: pod.Namespace,
			Status:    "Deleted",
		}
	case watch.Error:
		event.Type = EventError
		event.Metadata["error"] = "watch error"
	}

	if handler != nil {
		handler(event)
	}
}

// handleDeploymentEvent handles deployment change events
func (kc *KubernetesClient) handleDeploymentEvent(eventType watch.EventType, deployment *corev1.Deployment, handler func(*Event)) {
	event := &Event{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Map watch events to our event types
	switch eventType {
	case watch.Added:
		event.Type = EventDeploymentAdded
		event.Deployment = &ServiceResource{
			Type:      "deployment",
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Replicas:  int(*deployment.Spec.Replicas),
		}
	case watch.Modified:
		event.Type = EventDeploymentUpdated
		event.Deployment = &ServiceResource{
			Type:      "deployment",
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
			Replicas:  int(*deployment.Spec.Replicas),
		}
	case watch.Deleted:
		event.Type = EventDeploymentDeleted
		event.Deployment = &ServiceResource{
			Type:      "deployment",
			Name:      deployment.Name,
			Namespace: deployment.Namespace,
		}
	case watch.Error:
		event.Type = EventError
		event.Metadata["error"] = "watch error"
	}

	if handler != nil {
		handler(event)
	}
}

// handleServiceEvent handles service change events
func (kc *KubernetesClient) handleServiceEvent(eventType watch.EventType, service *corev1.Service, handler func(*Event)) {
	event := &Event{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Map watch events to our event types
	switch eventType {
	case watch.Added:
		event.Type = EventServiceAdded
		event.Service = &ServiceResource{
			Type:      "service",
			Name:      service.Name,
			Namespace: service.Namespace,
		}
	case watch.Modified:
		event.Type = EventServiceUpdated
		event.Service = &ServiceResource{
			Type:      "service",
			Name:      service.Name,
			Namespace: service.Namespace,
		}
	case watch.Deleted:
		event.Type = EventServiceDeleted
		event.Service = &ServiceResource{
			Type:      "service",
			Name:      service.Name,
			Namespace: service.Namespace,
		}
	case watch.Error:
		event.Type = EventError
		event.Metadata["error"] = "watch error"
	}

	if handler != nil {
		handler(event)
	}
}

// RegisterEventHandler registers a handler for specific event types
func (kc *KubernetesClient) RegisterEventHandler(eventType EventType, handler func(*Event)) {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	kc.eventHandlers[eventType] = append(kc.eventHandlers[eventType], handler)
}

// Shutdown stops the Kubernetes client
func (kc *KubernetesClient) Shutdown() {
	kc.mu.Lock()
	defer kc.mu.Unlock()

	if kc.cancel != nil {
		kc.cancel()
	}

	// Stop all watchers
	for name, controller := range kc.watchers {
		if controller != nil {
			controller.Stop()
			delete(kc.watchers, name)
			kc.logger.WithField("watcher", name).Info("Watcher stopped")
		}
	}

	kc.initialized = false

	kc.logger.Info("Kubernetes client shut down")
}

// GetClient returns the underlying Kubernetes clientset
func (kc *KubernetesClient) GetClient() *kubernetes.Clientset {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.clientset
}

// GetDynamicClient returns the dynamic client
func (kc *KubernetesClient) GetDynamicClient() *dynamic.DynamicClient {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.dynamicClient
}

// GetRestConfig returns the Kubernetes config
func (kc *KubernetesClient) GetRestConfig() *rest.Config {
	kc.mu.RLock()
	defer kc.mu.RUnlock()
	return kc.restCfg
}

// ListAllResources lists all Kubernetes resources of specified types
func (kc *KubernetesClient) ListAllResources(resourceTypes []string, namespaces []string) ([]*ServiceResource, error) {
	kc.mu.RLock()
	clientset := kc.clientset
	kc.mu.RUnlock()

	if clientset == nil {
		return nil, ErrK8sNotConnected
	}

	var resources []*ServiceResource

	// Default to all resource types if none specified
	if len(resourceTypes) == 0 {
		resourceTypes = []string{
			"pod",
			"deployment",
			"service",
			"statefulset",
			"daemonset",
			"replicaset",
			"configmap",
			"secret",
			"ingress",
			"persistentvolumeclaim",
			"namespace",
		}
	}

	// Default to all namespaces if none specified
	if len(namespaces) == 0 {
		namespaces = []string{v1.NamespaceAll}
	}

	ctx, cancel := context.WithTimeout(kc.ctx, 60*time.Second)
	defer cancel()

	for _, ns := range namespaces {
		for _, rt := range resourceTypes {
			resource, err := kc.collectResource(ctx, clientset, rt, ns)
			if err != nil {
				kc.logger.WithFields(logrus.Fields{
					"type": rt,
					"ns":   ns,
					"error": err,
				}).Debug("Failed to collect resource")
				continue
			}
			if resource != nil {
				resources = append(resources, resource)
			}
		}
	}

	return resources, nil
}

// collectResource collects a specific Kubernetes resource type
func (kc *KubernetesClient) collectResource(ctx context.Context, clientset *kubernetes.Clientset, resourceType, namespace string) (*ServiceResource, error) {
	switch resourceType {
	case "pod":
		return kc.collectPods(ctx, clientset, namespace)
	case "deployment":
		return kc.collectDeployments(ctx, clientset, namespace)
	case "statefulset":
		return kc.collectStatefulSets(ctx, clientset, namespace)
	case "daemonset":
		return kc.collectDaemonSets(ctx, clientset, namespace)
	case "replicaset":
		return kc.collectReplicaSets(ctx, clientset, namespace)
	case "service":
		return kc.collectServices(ctx, clientset, namespace)
	case "ingress":
		return kc.collectIngresses(ctx, clientset, namespace)
	case "configmap":
		return kc.collectConfigMaps(ctx, clientset, namespace)
	case "secret":
		return kc.collectSecrets(ctx, clientset, namespace)
	case "persistentvolumeclaim":
		return kc.collectPVCs(ctx, clientset, namespace)
	case "namespace":
		return kc.collectNamespaces(ctx, clientset, namespace)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// collectPods collects all pods in a namespace
func (kc *KubernetesClient) collectPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		return nil, nil
	}

	// Aggregate pod information
	readyPods := 0
	totalPods := len(pods.Items)

	for _, pod := range pods.Items {
		if pod.Status.Phase == v1.PodRunning {
			readyPods++
		}
	}

	return &ServiceResource{
		Type:      "pod",
		Name:      "pods",
		Namespace: namespace,
		Status:    string(v1.NamespaceRunning),
		Conditions: []ResourceCondition{
			{
				Type:   "Ready",
				Status: fmt.Sprintf("%d/%d", readyPods, totalPods),
			},
		},
	}, nil
}

// collectDeployments collects all deployments in a namespace
func (kc *KubernetesClient) collectDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(deployments.Items) == 0 {
		return nil, nil
	}

	// Return first deployment as representative
	dep := deployments.Items[0]
	return &ServiceResource{
		Type:      "deployment",
		Name:      dep.Name,
		Namespace: namespace,
		Replicas:  int(*dep.Spec.Replicas),
		ReadyReplicas: int(dep.Status.ReadyReplicas),
	}, nil
}

// collectStatefulSets collects all stateful sets in a namespace
func (kc *KubernetesClient) collectStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(statefulsets.Items) == 0 {
		return nil, nil
	}

	ss := statefulsets.Items[0]
	return &ServiceResource{
		Type:      "statefulset",
		Name:      ss.Name,
		Namespace: namespace,
		Replicas:  int(ss.Status.Replicas),
		ReadyReplicas: int(ss.Status.ReadyReplicas),
	}, nil
}

// collectDaemonSets collects all daemon sets in a namespace
func (kc *KubernetesClient) collectDaemonSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(daemonsets.Items) == 0 {
		return nil, nil
	}

	ds := daemonsets.Items[0]
	return &ServiceResource{
		Type:      "daemonset",
		Name:      ds.Name,
		Namespace: namespace,
		ReadyReplicas: int(ds.Status.NumberReady),
	}, nil
}

// collectReplicaSets collects all replica sets in a namespace
func (kc *KubernetesClient) collectReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	replicaSets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(replicaSets.Items) == 0 {
		return nil, nil
	}

	rs := replicaSets.Items[0]
	return &ServiceResource{
		Type:      "replicaset",
		Name:      rs.Name,
		Namespace: namespace,
		Replicas:  int(rs.Status.Replicas),
		ReadyReplicas: int(rs.Status.ReadyReplicas),
	}, nil
}

// collectServices collects all services in a namespace
func (kc *KubernetesClient) collectServices(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(services.Items) == 0 {
		return nil, nil
	}

	return &ServiceResource{
		Type:      "service",
		Name:      services.Items[0].Name,
		Namespace: namespace,
	}, nil
}

// collectIngresses collects all ingresses in a namespace
func (kc *KubernetesClient) collectIngresses(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(ingresses.Items) == 0 {
		return nil, nil
	}

	return &ServiceResource{
		Type:      "ingress",
		Name:      ingresses.Items[0].Name,
		Namespace: namespace,
	}, nil
}

// collectConfigMaps collects all configmaps in a namespace
func (kc *KubernetesClient) collectConfigMaps(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	configmaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(configmaps.Items) == 0 {
		return nil, nil
	}

	return &ServiceResource{
		Type:      "configmap",
		Name:      configmaps.Items[0].Name,
		Namespace: namespace,
	}, nil
}

// collectSecrets collects all secrets in a namespace
func (kc *KubernetesClient) collectSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(secrets.Items) == 0 {
		return nil, nil
	}

	return &ServiceResource{
		Type:      "secret",
		Name:      secrets.Items[0].Name,
		Namespace: namespace,
	}, nil
}

// collectPVCs collects all PVCs in a namespace
func (kc *KubernetesClient) collectPVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(pvcs.Items) == 0 {
		return nil, nil
	}

	return &ServiceResource{
		Type:      "persistentvolumeclaim",
		Name:      pvcs.Items[0].Name,
		Namespace: namespace,
	}, nil
}

// collectNamespaces collects all namespaces
func (kc *KubernetesClient) collectNamespaces(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	if len(namespaces.Items) == 0 {
		return nil, nil
	}

	// Return first namespace as representative
	ns := namespaces.Items[0]
	return &ServiceResource{
		Type:      "namespace",
		Name:      ns.Name,
		Namespace: "kube-system",
	}, nil
}
