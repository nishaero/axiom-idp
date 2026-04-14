//go:build ignore

package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// DockerMetricsCollector implements Docker API integration
type DockerMetricsCollector struct {
	mu       sync.RWMutex
	client   *client.Client
	config   *DiscoveryConfig
	logger   *logrus.Logger
	ctx      context.Context
	cancel   context.CancelFunc
}

// KubernetesMetricsCollector implements Kubernetes API integration
type KubernetesMetricsCollector struct {
	mu        sync.RWMutex
	clientset *kubernetes.Clientset
	config    *DiscoveryConfig
	logger    *logrus.Logger
	restCfg   *rest.Config
	ctx       context.Context
	cancel    context.CancelFunc
}

// CombinedMetricsCollector combines Docker and Kubernetes metrics
type CombinedMetricsCollector struct {
	docker  *DockerMetricsCollector
	k8s     *KubernetesMetricsCollector
	config  *DiscoveryConfig
	logger  *logrus.Logger
	mu      sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector based on configuration
func NewMetricsCollector(cfg *DiscoveryConfig) MetricsCollector {
	combined := &CombinedMetricsCollector{
		config: cfg,
		logger: logrus.New(),
	}

	combined.logger.SetLevel(logrus.InfoLevel)
	combined.logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Initialize Docker collector if enabled
	if cfg.EnableDocker {
		combined.docker = NewDockerMetricsCollector(cfg, combined.logger)
	}

	// Initialize Kubernetes collector if enabled
	if cfg.EnableKubernetes {
		combined.k8s = NewKubernetesMetricsCollector(cfg, combined.logger)
	}

	return combined
}

// SetConfig updates the configuration
func (c *CombinedMetricsCollector) SetConfig(cfg *DiscoveryConfig) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config = cfg
	if c.docker != nil {
		c.docker.SetConfig(cfg)
	}
	if c.k8s != nil {
		c.k8s.SetConfig(cfg)
	}
}

// CollectionName returns the collection name
func (c *CombinedMetricsCollector) CollectionName() string {
	return "combined"
}

// CollectContainers collects container information from Docker
func (c *CombinedMetricsCollector) CollectContainers() ([]*Container, error) {
	if c.docker == nil {
		return nil, nil
	}
	return c.docker.CollectContainers()
}

// CollectImages collects image information from Docker
func (c *CombinedMetricsCollector) CollectImages() ([]*Image, error) {
	if c.docker == nil {
		return nil, nil
	}
	return c.docker.CollectImages()
}

// CollectNetworks collects network information from Docker
func (c *CombinedMetricsCollector) CollectNetworks() ([]*Network, error) {
	if c.docker == nil {
		return nil, nil
	}
	return c.docker.CollectNetworks()
}

// CollectKubernetesResources collects Kubernetes resources
func (c *CombinedMetricsCollector) CollectKubernetesResources() ([]*ServiceResource, error) {
	if c.k8s == nil {
		return nil, nil
	}
	return c.k8s.CollectKubernetesResources()
}

// NewDockerMetricsCollector creates a new Docker metrics collector
func NewDockerMetricsCollector(cfg *DiscoveryConfig, logger *logrus.Logger) *DockerMetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &DockerMetricsCollector{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetConfig updates the configuration
func (d *DockerMetricsCollector) SetConfig(cfg *DiscoveryConfig) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config = cfg
}

// CollectionName returns the collection name
func (d *DockerMetricsCollector) CollectionName() string {
	return "docker"
}

// CollectContainers collects container information from Docker daemon
func (d *DockerMetricsCollector) CollectContainers() ([]*Container, error) {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	if client == nil {
		// Initialize client if not already initialized
		if err := d.initializeClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize Docker client: %w", err)
		}
		client = d.client
	}

	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	// Get container list
	containers, err := client.ContainerList(ctx, container.ListOptions{
		All: true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []*Container
	for _, c := range containers {
		containerInfo, err := client.ContainerInspect(ctx, c.ID)
		if err != nil {
			d.logger.WithFields(logrus.Fields{
				"container": c.Names,
				"error":     err,
			}).Error("Failed to inspect container")
			continue
		}

		stats, err := client.ContainerStats(ctx, c.ID, false)
		if err != nil {
			d.logger.WithFields(logrus.Fields{
				"container": c.Names,
				"error":     err,
			}).Debug("Failed to get container stats")
		}

		// Parse stats
		var statsData *ContainerStats
		if stats != nil && stats.Body != nil {
			var rawStats map[string]interface{}
			if err := json.NewDecoder(stats.Body).Decode(&rawStats); err == nil {
				statsData = parseContainerStats(rawStats)
			}
			stats.Body.Close()
		}

		// Determine status
		status := parseContainerStatus(containerInfo.State)

		// Parse networks
		networks := parseContainerNetworks(containerInfo.NetworkSettings)

		// Parse health status
		var health *Health
		if containerInfo.State.Health != nil {
			health = parseHealthStatus(containerInfo.State.Health)
		}

		// Extract labels and environment
		labels := make(map[string]string)
		for k, v := range containerInfo.Config.Labels {
			labels[k] = v
		}

		environment := make(map[string]string)
		for _, env := range containerInfo.Config.Env {
			if parts := strings.SplitN(env, "=", 2); len(parts) == 2 {
				environment[parts[0]] = parts[1]
			}
		}

		result = append(result, &Container{
			ID:            containerInfo.ID,
			Name:          strings.TrimPrefix(containerInfo.Name, "/"),
			Image:         containerConfig.Image,
			Status:        status,
			State:         containerInfo.State.Status,
			Ports:         parsePortMappings(containerInfo.NetworkSettings.Ports),
			Networks:      networks,
			Labels:        labels,
			Environment:   environment,
			Created:       containerInfo.Created,
			StartedAt:     containerInfo.StartedAt,
			HealthStatus:  containerInfo.State.Health.Status,
			Health:        health,
			Host:          "localhost",
			SysInitPresent: true,
			OpenStdin:     containerInfo.Config.OpenStdin,
			StdinOnce:     containerInfo.Config.StdinOnce,
			Isolation:     string(containerInfo.HostConfig.Isolation),
			Stats:         statsData,
		})
	}

	d.logger.WithFields(logrus.Fields{
		"count": len(result),
	}).Info("Docker container collection completed")

	return result, nil
}

// CollectImages collects image information from Docker
func (d *DockerMetricsCollector) CollectImages() ([]*Image, error) {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	if client == nil {
		if err := d.initializeClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize Docker client: %w", err)
		}
		client = d.client
	}

	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	// Get image list
	images, err := client.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	var result []*Image
	for _, img := range images {
		// Get detailed image info
		detailed, _, err := client.ImageInspectWithRaw(ctx, img.ID)
		if err != nil {
			d.logger.WithFields(logrus.Fields{
				"image": img.ID,
				"error": err,
			}).Debug("Failed to inspect image")
			continue
		}

		// Extract labels
		labels := make(map[string]string)
		for k, v := range detailed.Config.Labels {
			labels[k] = v
		}

		result = append(result, &Image{
			ID:            img.ID,
			RepoTags:      img.RepoTags,
			Created:       time.Unix(img.Created, 0),
			Size:          img.Size,
			VirtualSize:   img.VirtualSize,
			Labels:        labels,
			Digest:        img.Digest,
			Architecture:  detailed.Architecture,
			Os:            detailed.OS,
		})
	}

	d.logger.WithFields(logrus.Fields{
		"count": len(result),
	}).Info("Docker image collection completed")

	return result, nil
}

// CollectNetworks collects network information from Docker
func (d *DockerMetricsCollector) CollectNetworks() ([]*Network, error) {
	d.mu.RLock()
	client := d.client
	d.mu.RUnlock()

	if client == nil {
		if err := d.initializeClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize Docker client: %w", err)
		}
		client = d.client
	}

	ctx, cancel := context.WithTimeout(d.ctx, 30*time.Second)
	defer cancel()

	// Get network list
	networks, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	var result []*Network
	for _, net := range networks {
		// Parse IPAM configuration
		var ipam NetworkIPAM
		if net.IPAM != nil {
			ipam = NetworkIPAM{
				Driver:  net.IPAM.Config,
				Config:  make([]IPAMConfig, 0),
				Options: net.IPAM.Options,
			}

			for _, config := range net.IPAM.Config {
				ipam.Config = append(ipam.Config, IPAMConfig{
					Subnet:  config.Subnet,
					IPRange: config.IPRange,
					Gateway: config.Gateway,
				})
			}
		}

		// Parse containers
		containers := make(map[string]*NetworkContainer)
		for id, containerInfo := range net.Containers {
			containers[id] = &NetworkContainer{
				Name:        containerInfo.Name,
				ID:          id,
				IPv4Address: containerInfo.IPv4Address,
				IPv6Address: containerInfo.IPv6Address,
			}
		}

		// Parse labels
		labels := make(map[string]string)
		for k, v := range net.Labels {
			labels[k] = v
		}

		result = append(result, &Network{
			Name:        net.Name,
			ID:          net.ID,
			Scope:       net.Scope,
			Type:        net.Type,
			Driver:      net.Driver,
			IPAM:        ipam,
			Labels:      labels,
			Containers:  containers,
			Created:     net.Created,
			Internal:    net.Internal,
			EnableIPv6:  net.EnableIPv6,
			Attachable:  net.Attachable,
			Ingress:     net.Ingress,
		})
	}

	d.logger.WithFields(logrus.Fields{
		"count": len(result),
	}).Info("Docker network collection completed")

	return result, nil
}

// initializeClient initializes the Docker client
func (d *DockerMetricsCollector) initializeClient() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client != nil {
		return nil
	}

	// Use default socket or configured socket
	socket := d.config.DockerSocket
	if socket == "" {
		socket = "unix:///var/run/docker.sock"
	}

	// Extract socket path
	var socketPath string
	if strings.HasPrefix(socket, "unix://") {
		socketPath = socket[7:]
	} else {
		socketPath = socket
	}

	// Create Docker client
	cli, err := client.NewClientWithOpts(
		client.WithAPINegotiation(),
		client.WithHost(fmt.Sprintf("unix://%s", socketPath)),
		client.WithVersion("1.41"),
	)
	if err != nil {
		return err
	}

	d.client = cli
	d.logger.WithFields(logrus.Fields{
		"socket": socket,
	}).Info("Docker client initialized")

	return nil
}

// NewKubernetesMetricsCollector creates a new Kubernetes metrics collector
func NewKubernetesMetricsCollector(cfg *DiscoveryConfig, logger *logrus.Logger) *KubernetesMetricsCollector {
	ctx, cancel := context.WithCancel(context.Background())

	return &KubernetesMetricsCollector{
		config: cfg,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
	}
}

// SetConfig updates the configuration
func (k *KubernetesMetricsCollector) SetConfig(cfg *DiscoveryConfig) {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.config = cfg
}

// CollectionName returns the collection name
func (k *KubernetesMetricsCollector) CollectionName() string {
	return "kubernetes"
}

// CollectKubernetesResources collects Kubernetes resources based on configuration
func (k *KubernetesMetricsCollector) CollectKubernetesResources() ([]*ServiceResource, error) {
	k.mu.RLock()
	clientset := k.clientset
	config := k.config
	k.mu.RUnlock()

	if clientset == nil {
		if err := k.initializeClient(); err != nil {
			return nil, fmt.Errorf("failed to initialize Kubernetes client: %w", err)
		}
		clientset = k.clientset
	}

	ctx, cancel := context.WithTimeout(k.ctx, 60*time.Second)
	defer cancel()

	var resources []*ServiceResource
	var errors []error

	// Collect based on resource types
	resourceTypes := config.ResourceTypes
	if len(resourceTypes) == 0 {
		resourceTypes = []string{
			ResourceTypePod,
			ResourceTypeDeployment,
			ResourceTypeService,
			ResourceTypeStatefulSet,
			ResourceTypeDaemonSet,
			ResourceTypeReplicaSet,
			ResourceTypeConfigMap,
			ResourceTypeSecret,
			ResourceTypeIngress,
			ResourceTypePersistentVolumeClaim,
			ResourceTypeNamespace,
		}
	}

	namespaces := config.Namespaces
	if len(namespaces) == 0 {
		namespaces = []string{v1.NamespaceAll}
	}

	for _, ns := range namespaces {
		for _, rt := range resourceTypes {
			resource, err := k.collectResource(ctx, clientset, rt, ns)
			if err != nil {
				errors = append(errors, err)
				k.logger.WithFields(logrus.Fields{
					"type":   rt,
					"ns":     ns,
					"error":  err,
				}).Debug("Failed to collect resource")
				continue
			}
			resources = append(resources, resource)
		}
	}

	if len(errors) > 0 {
		k.logger.WithFields(logrus.Fields{
			"collected": len(resources),
			"errors":    len(errors),
		}).Warn("Kubernetes collection completed with errors")
	} else {
		k.logger.WithFields(logrus.Fields{
			"collected": len(resources),
		}).Info("Kubernetes collection completed")
	}

	return resources, nil
}

// collectResource collects a specific Kubernetes resource type
func (k *KubernetesMetricsCollector) collectResource(ctx context.Context, clientset *kubernetes.Clientset, resourceType, namespace string) (*ServiceResource, error) {
	switch resourceType {
	case ResourceTypePod:
		return k.collectPods(ctx, clientset, namespace)
	case ResourceTypeDeployment:
		return k.collectDeployments(ctx, clientset, namespace)
	case ResourceTypeStatefulSet:
		return k.collectStatefulSets(ctx, clientset, namespace)
	case ResourceTypeDaemonSet:
		return k.collectDaemonSets(ctx, clientset, namespace)
	case ResourceTypeReplicaSet:
		return k.collectReplicaSets(ctx, clientset, namespace)
	case ResourceTypeService:
		return k.collectServices(ctx, clientset, namespace)
	case ResourceTypeIngress:
		return k.collectIngresses(ctx, clientset, namespace)
	case ResourceTypeConfigMap:
		return k.collectConfigMaps(ctx, clientset, namespace)
	case ResourceTypeSecret:
		return k.collectSecrets(ctx, clientset, namespace)
	case ResourceTypePersistentVolumeClaim:
		return k.collectPVCs(ctx, clientset, namespace)
	case ResourceTypeNamespace:
		return k.collectNamespaces(ctx, clientset)
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}
}

// collectPods collects all pods in a namespace
func (k *KubernetesMetricsCollector) collectPods(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	// Convert to map for quicker access
	podMap := make(map[string]*PodDetail)

	for _, pod := range pods.Items {
		// Extract conditions
		conditions := make([]ResourceCondition, 0)
		for _, cond := range pod.Status.Conditions {
			conditions = append(conditions, ResourceCondition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
			})
		}

		// Extract containers
		containers := make([]ContainerDetail, 0)
		restartCount := 0
		for _, c := range pod.Status.ContainerStatuses {
			restartCount += int(c.RestartCount)
			container := ContainerDetail{
				Name:   c.Name,
				Image:  c.Image,
				State:  string(c.State),
				Ready:  c.Ready,
				Restarts: int(c.RestartCount),
			}
			containers = append(containers, container)
		}

		// Extract owner references
		ownerRefs := make([]OwnerReference, 0)
		for _, owner := range pod.OwnerReferences {
			ownerRefs = append(ownerRefs, OwnerReference{
				Kind:       owner.Kind,
				Name:       owner.Name,
				UID:        string(owner.UID),
				APIVersion: owner.APIVersion,
			})
		}

		// Get controller name and ID
		var controller, controllerID string
		if len(ownerRefs) > 0 {
			controller = ownerRefs[0].Kind
			controllerID = ownerRefs[0].UID
		}

		podDetail := &PodDetail{
			Name:        pod.Name,
			Status:      string(pod.Status.Phase),
			Ready:       fmt.Sprintf("%d/%d", pod.Status.ReadyReplicas, pod.Status.Replicas),
			Restarts:    restartCount,
			PodIP:       pod.Status.PodIP,
			Node:        pod.Spec.NodeName,
			Labels:      pod.Labels,
			Annotations: pod.Annotations,
			Containers:  containers,
			OwnerRefs:   ownerRefs,
			Conditions:  conditions,
		}

		podMap[pod.Name] = podDetail
	}

	return &ServiceResource{
		Type:      Resource("pod"),
		Name:      "pods",
		Namespace: namespace,
		Status:    string(v1.NamespaceRunning),
		Conditions: []ResourceCondition{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}, nil
}

// collectDeployments collects all deployments in a namespace
func (k *KubernetesMetricsCollector) collectDeployments(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, deployment := range deployments.Items {
		// Extract conditions
		conditions := make([]ResourceCondition, 0)
		for _, cond := range deployment.Status.Conditions {
			conditions = append(conditions, ResourceCondition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
			})
		}

		// Extract labels and annotations
		labels := deployment.Labels
		annotations := deployment.Annotations

		resources = append(resources, &ServiceResource{
			Type:      Resource("deployment"),
			Name:      deployment.Name,
			Namespace: namespace,
			Labels:    labels,
			Annotations: annotations,
			Status:    string(deployment.Status.Phase),
			Replicas:  int(*deployment.Spec.Replicas),
			ReadyReplicas: int(deployment.Status.ReadyReplicas),
			Conditions: conditions,
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectStatefulSets collects all stateful sets in a namespace
func (k *KubernetesMetricsCollector) collectStatefulSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	statefulsets, err := clientset.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, ss := range statefulsets.Items {
		conditions := make([]ResourceCondition, 0)
		for _, cond := range ss.Status.Conditions {
			conditions = append(conditions, ResourceCondition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
			})
		}

		labels := ss.Labels
		annotations := ss.Annotations

		resources = append(resources, &ServiceResource{
			Type:          Resource("statefulset"),
			Name:          ss.Name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   annotations,
			Status:        string(ss.Status.Phase),
			Replicas:      int(ss.Status.Replicas),
			ReadyReplicas: int(ss.Status.ReadyReplicas),
			Conditions:    conditions,
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectDaemonSets collects all daemon sets in a namespace
func (k *KubernetesMetricsCollector) collectDaemonSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, ds := range daemonsets.Items {
		conditions := make([]ResourceCondition, 0)
		for _, cond := range ds.Status.Conditions {
			conditions = append(conditions, ResourceCondition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
			})
		}

		labels := ds.Labels
		annotations := ds.Annotations

		resources = append(resources, &ServiceResource{
			Type:                   "daemonset",
			Name:                   ds.Name,
			Namespace:              namespace,
			Labels:                 labels,
			Annotations:            annotations,
			Status:                 string(ds.Status.Phase),
			ReadyReplicas:          int(ds.Status.NumberReady),
			CurrentNumberScheduled: int(ds.Status.CurrentNumberScheduled),
			DesiredNumberScheduled: int(ds.Status.DesiredNumberScheduled),
			Conditions:             conditions,
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectReplicaSets collects all replica sets in a namespace
func (k *KubernetesMetricsCollector) collectReplicaSets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	replicasets, err := clientset.AppsV1().ReplicaSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, rs := range replicasets.Items {
		conditions := make([]ResourceCondition, 0)
		for _, cond := range rs.Status.Conditions {
			conditions = append(conditions, ResourceCondition{
				Type:    string(cond.Type),
				Status:  string(cond.Status),
			})
		}

		labels := rs.Labels
		annotations := rs.Annotations

		resources = append(resources, &ServiceResource{
			Type:              "replicaset",
			Name:              rs.Name,
			Namespace:         namespace,
			Labels:            labels,
			Annotations:       annotations,
			Status:            string(rs.Status.Phase),
			Replicas:          int(rs.Status.Replicas),
			AvailableReplicas: int(rs.Status.AvailableReplicas),
			ReadyReplicas:     int(rs.Status.ReadyReplicas),
			Conditions:        conditions,
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectServices collects all services in a namespace
func (k *KubernetesMetricsCollector) collectServices(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, svc := range services.Items {
		labels := svc.Labels
		annotations := svc.Annotations

		resources = append(resources, &ServiceResource{
			Type:          "service",
			Name:          svc.Name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   annotations,
			Status:        string(svc.Status.Phase),
			Conditions: []ResourceCondition{
				{
					Type:   "LoadedBalancerIngress",
					Status: string(v1.ConditionTrue),
				},
			},
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectIngresses collects all ingresses in a namespace
func (k *KubernetesMetricsCollector) collectIngresses(ctx context.Context, clientset *kubernetes.Interface, namespace string) (*ServiceResource, error) {
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, ing := range ingresses.Items {
		labels := ing.Labels
		annotations := ing.Annotations

		resources = append(resources, &ServiceResource{
			Type:          "ingress",
			Name:          ing.Name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   annotations,
			Status:        string(v1.NamespaceRunning),
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectConfigMaps collects all config maps in a namespace
func (k *KubernetesMetricsCollector) collectConfigMaps(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, cm := range configMaps.Items {
		labels := cm.Labels
		annotations := cm.Annotations

		resources = append(resources, &ServiceResource{
			Type:          "configmap",
			Name:          cm.Name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   annotations,
			Status:        string(v1.NamespaceRunning),
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectSecrets collects all secrets in a namespace
func (k *KubernetesMetricsCollector) collectSecrets(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	secrets, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, secret := range secrets.Items {
		labels := secret.Labels
		annotations := secret.Annotations

		resources = append(resources, &ServiceResource{
			Type:          "secret",
			Name:          secret.Name,
			Namespace:     namespace,
			Labels:        labels,
			Annotations:   annotations,
			Status:        string(v1.NamespaceRunning),
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectPVCs collects all persistent volume claims in a namespace
func (k *KubernetesMetricsCollector) collectPVCs(ctx context.Context, clientset *kubernetes.Clientset, namespace string) (*ServiceResource, error) {
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, pvc := range pvcs.Items {
		labels := pvc.Labels
		annotations := pvc.Annotations

		resources = append(resources, &ServiceResource{
			Type:               "persistentvolumeclaim",
			Name:               pvc.Name,
			Namespace:          namespace,
			Labels:             labels,
			Annotations:        annotations,
			Status:             string(pvc.Status.Phase),
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// collectNamespaces collects all namespaces
func (k *KubernetesMetricsCollector) collectNamespaces(ctx context.Context, clientset *kubernetes.Clientset) (*ServiceResource, error) {
	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var resources []*ServiceResource

	for _, ns := range namespaces.Items {
		labels := ns.Labels
		annotations := ns.Annotations

		resources = append(resources, &ServiceResource{
			Type:      "namespace",
			Name:      ns.Name,
			Namespace: "kube-system",
			Labels:    labels,
			Annotations: annotations,
			Status:    string(ns.Status.Phase),
		})
	}

	if len(resources) == 0 {
		return nil, nil
	}

	return resources[0], nil
}

// initializeClient initializes the Kubernetes client
func (k *KubernetesMetricsCollector) initializeClient() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if k.clientset != nil {
		return nil
	}

	var err error

	// Try in-cluster config first
	k.restCfg, err = rest.InClusterConfig()
	if err == nil {
		k.clientset, err = kubernetes.NewForConfig(k.restCfg)
		if err == nil {
			k.logger.Info("Connected to Kubernetes cluster (in-cluster)")
			return nil
		}
	}

	// Fall back to kubeconfig file
	kubeconfigPath := k.config.KubeconfigPath
	if kubeconfigPath == "" {
		// Default to $HOME/.kube/config
		home := os.Getenv("HOME")
		if home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfigPath != "" {
		k.restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err == nil {
			k.clientset, err = kubernetes.NewForConfig(k.restCfg)
			if err == nil {
				k.logger.WithFields(logrus.Fields{
					"config": kubeconfigPath,
				}).Info("Connected to Kubernetes cluster (kubeconfig)")
				return nil
			}
		}
	}

	// Try default config location
	k.restCfg, err = clientcmd.LoadFromFile(clientcmd.RecommendedHomeFile)
	if err != nil {
		k.logger.WithError(err).Error("Failed to load Kubernetes configuration")
		return fmt.Errorf("failed to initialize Kubernetes client: %w", err)
	}

	k.clientset, err = kubernetes.NewForConfig(k.restCfg)
	if err != nil {
		k.logger.WithError(err).Error("Failed to create Kubernetes client")
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	k.logger.Info("Connected to Kubernetes cluster")
	return nil
}
