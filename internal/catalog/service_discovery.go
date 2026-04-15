//go:build ignore

package catalog

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	networkingv1 "k8s.io/api/networking/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServiceDiscovery provides comprehensive service discovery functionality
type ServiceDiscovery struct {
	mu           sync.RWMutex
	dockerClient *DockerClient
	k8sClient    *KubernetesClient
	eventBus     *EventBus
	eventStream  *RedisEventStream
	hub          *WebSocketHub
	config       *DiscoveryConfig
	logger       *logrus.Logger
	ctx          context.Context
	cancel       context.CancelFunc
	initialized  bool
	hotReload    bool
	lastDiscovery time.Time
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(cfg *DiscoveryConfig, logger *logrus.Logger) *ServiceDiscovery {
	ctx, cancel := context.WithCancel(context.Background())

	sd := &ServiceDiscovery{
		config:      cfg,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		initialized: false,
		hotReload:   cfg.EnableHotReload,
	}

	// Initialize components based on configuration
	if cfg.EnableDocker {
		sd.dockerClient = NewDockerClient(cfg, logger)
	}

	if cfg.EnableKubernetes {
		sd.k8sClient = NewKubernetesClient(cfg, logger)
	}

	if cfg.EnableEventBus {
		sd.eventBus = NewEventBus()
	}

	if cfg.EnableRedisStream {
		redisConfig := &RedisConfig{
			Host:            cfg.RedisHost,
			Port:            cfg.RedisPort,
			Password:        cfg.RedisPassword,
			DB:              cfg.RedisDB,
			EnableStreaming: cfg.EnableEventStreaming,
			Prefix:          "axiom",
			StreamName:      "service-events",
			ConsumerGroup:   "axiom-consumers",
			ConsumerName:    "discovery-consumer",
			MaxStreamLength: 10000,
			PollInterval:    1 * time.Second,
		}
		sd.eventStream = NewRedisEventStream(redisConfig, logger)
	}

	if cfg.EnableWebSocket {
		sd.hub = NewWebSocketHub(logger)
	}

	logger.WithFields(logrus.Fields{
		"docker":       cfg.EnableDocker,
		"kubernetes":   cfg.EnableKubernetes,
		"eventBus":     cfg.EnableEventBus,
		"redisStream":  cfg.EnableRedisStream,
		"webSocket":    cfg.EnableWebSocket,
		"hotReload":    cfg.EnableHotReload,
	}).Info("Service Discovery initialized")

	return sd
}

// Initialize initializes all configured components
func (sd *ServiceDiscovery) Initialize() error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	var errs []error

	if sd.config.EnableDocker {
		if err := sd.dockerClient.initializeClient(); err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize Docker client: %w", err))
		}
	}

	if sd.config.EnableKubernetes {
		if err := sd.k8sClient.initializeClient(); err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize Kubernetes client: %w", err))
		}
	}

	if sd.config.EnableRedisStream {
		if err := sd.eventStream.initialize(); err != nil {
			errs = append(errs, fmt.Errorf("failed to initialize Redis stream: %w", err))
		}

		if sd.config.EnableEventStreaming {
			sd.eventStream.StartConsumer("discovery-consumer", sd.handleStreamEvent)
		}
	}

	if sd.config.EnableWebSocket {
		sd.hub.Start()
	}

	sd.initialized = true

	if len(errs) > 0 {
		return fmt.Errorf("initialization completed with errors: %v", errs)
	}

	sd.logger.Info("Service Discovery initialized successfully")
	return nil
}

// InitializeDockerClient initializes only the Docker client
func (sd *ServiceDiscovery) InitializeDockerClient() error {
	if sd.dockerClient == nil {
		sd.dockerClient = NewDockerClient(sd.config, sd.logger)
	}
	return sd.dockerClient.initializeClient()
}

// InitializeKubernetesClient initializes only the Kubernetes client
func (sd *ServiceDiscovery) InitializeKubernetesClient() error {
	if sd.k8sClient == nil {
		sd.k8sClient = NewKubernetesClient(sd.config, sd.logger)
	}
	return sd.k8sClient.initializeClient()
}

// SetConfig updates the configuration
func (sd *ServiceDiscovery) SetConfig(cfg *DiscoveryConfig) {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.config = cfg

	// Update clients if they exist
	if sd.dockerClient != nil {
		sd.dockerClient.SetConfig(cfg)
	}
	if sd.k8sClient != nil {
		sd.k8sClient.SetConfig(cfg)
	}

	// Enable/disable hot reload
	sd.hotReload = cfg.EnableHotReload
}

// GetConfig returns the current configuration
func (sd *ServiceDiscovery) GetConfig() *DiscoveryConfig {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.config
}

// GetDockerClient returns the Docker client
func (sd *ServiceDiscovery) GetDockerClient() *DockerClient {
	return sd.dockerClient
}

// GetKubernetesClient returns the Kubernetes client
func (sd *ServiceDiscovery) GetKubernetesClient() *KubernetesClient {
	return sd.k8sClient
}

// GetEventBus returns the event bus
func (sd *ServiceDiscovery) GetEventBus() *EventBus {
	return sd.eventBus
}

// GetRedisEventStream returns the Redis event stream
func (sd *ServiceDiscovery) GetRedisEventStream() *RedisEventStream {
	return sd.eventStream
}

// GetWebSocketHub returns the WebSocket hub
func (sd *ServiceDiscovery) GetWebSocketHub() *WebSocketHub {
	return sd.hub
}

// IsInitialized returns whether the service discovery is initialized
func (sd *ServiceDiscovery) IsInitialized() bool {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.initialized
}

// IsHealthy returns whether the service discovery is healthy
func (sd *ServiceDiscovery) IsHealthy() bool {
	if sd.dockerClient != nil && sd.dockerClient.IsConnected() {
		return true
	}

	if sd.k8sClient != nil && sd.k8sClient.IsConnected() {
		return true
	}

	return false
}

// DiscoverServices performs a full service discovery
func (sd *ServiceDiscovery) DiscoverServices() ([]*Service, error) {
	sd.mu.Lock()
	sd.lastDiscovery = time.Now()
	sd.mu.Unlock()

	var services []*Service

	if sd.dockerClient != nil && sd.dockerClient.IsConnected() {
		containers, err := sd.dockerClient.ListContainers(true)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to list Docker containers")
		} else {
			for _, c := range containers {
				svc, err := sd.convertDockerContainerToService(&c)
				if err != nil {
					sd.logger.WithError(err).Error("Failed to convert container to service")
					continue
				}
				services = append(services, svc)
			}
		}
	}

	if sd.k8sClient != nil && sd.k8sClient.IsConnected() {
		resources, err := sd.k8sClient.ListAllResources(sd.config.ResourceTypes, sd.config.Namespaces)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to list Kubernetes resources")
		} else {
			for _, r := range resources {
				svc := &Service{
					Name:            r.Name,
					Type:            "kubernetes",
					Labels:          r.Labels,
					Annotations:     r.Annotations,
					HealthStatus:    "unknown",
					LastUpdated:     time.Now(),
				}
				services = append(services, svc)
			}
		}
	}

	// Publish discovery event
	event := &Event{
		Type:      EventDiscoveryCompleted,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"services_found": len(services),
			"source":         "full_discovery",
		},
	}

	if sd.eventBus != nil {
		sd.eventBus.Publish(event)
	}

	if sd.eventStream != nil {
		sd.eventStream.PublishEvent(event)
	}

	sd.logger.WithField("services", len(services)).Info("Service discovery completed")

	return services, nil
}

// DiscoverContainers discovers Docker containers
func (sd *ServiceDiscovery) DiscoverContainers() ([]*Container, error) {
	if sd.dockerClient == nil {
		return nil, ErrDockerNotConnected
	}

	containers, err := sd.dockerClient.ListContainers(true)
	if err != nil {
		return nil, err
	}

	var result []*Container
	for _, c := range containers {
		container, err := sd.convertDockerContainerToContainer(&c)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to convert container")
			continue
		}
		result = append(result, container)
	}

	return result, nil
}

// DiscoverPods discovers Kubernetes pods
func (sd *ServiceDiscovery) DiscoverPods() ([]*v1.Pod, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allPods []*v1.Pod
	for _, ns := range sd.config.Namespaces {
		pods, err := sd.k8sClient.ListPods(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list pods in namespace %s", ns)
			continue
		}
		allPods = append(allPods, pods...)
	}

	return allPods, nil
}

// DiscoverServicesList discovers Kubernetes services
func (sd *ServiceDiscovery) DiscoverServicesList() ([]*v1.Service, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allServices []*v1.Service
	for _, ns := range sd.config.Namespaces {
		services, err := sd.k8sClient.ListServices(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list services in namespace %s", ns)
			continue
		}
		allServices = append(allServices, services...)
	}

	return allServices, nil
}

// DiscoverDeployments discovers Kubernetes deployments
func (sd *ServiceDiscovery) DiscoverDeployments() ([]*v1.Deployment, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allDeployments []*v1.Deployment
	for _, ns := range sd.config.Namespaces {
		deployments, err := sd.k8sClient.ListDeployments(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list deployments in namespace %s", ns)
			continue
		}
		allDeployments = append(allDeployments, deployments...)
	}

	return allDeployments, nil
}

// DiscoverConfigMaps discovers Kubernetes configmaps
func (sd *ServiceDiscovery) DiscoverConfigMaps() ([]*v1.ConfigMap, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allConfigMaps []*v1.ConfigMap
	for _, ns := range sd.config.Namespaces {
		configmaps, err := sd.k8sClient.ListConfigMaps(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list configmaps in namespace %s", ns)
			continue
		}
		allConfigMaps = append(allConfigMaps, configmaps...)
	}

	return allConfigMaps, nil
}

// DiscoverSecrets discovers Kubernetes secrets
func (sd *ServiceDiscovery) DiscoverSecrets() ([]*v1.Secret, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allSecrets []*v1.Secret
	for _, ns := range sd.config.Namespaces {
		secrets, err := sd.k8sClient.ListSecrets(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list secrets in namespace %s", ns)
			continue
		}
		allSecrets = append(allSecrets, secrets...)
	}

	return allSecrets, nil
}

// DiscoverStatefulSets discovers Kubernetes stateful sets
func (sd *ServiceDiscovery) DiscoverStatefulSets() ([]*v1.StatefulSet, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allStatefulSets []*v1.StatefulSet
	for _, ns := range sd.config.Namespaces {
		statefulsets, err := sd.k8sClient.ListStatefulSets(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list statefulsets in namespace %s", ns)
			continue
		}
		allStatefulSets = append(allStatefulSets, statefulsets...)
	}

	return allStatefulSets, nil
}

// DiscoverDaemonSets discovers Kubernetes daemon sets
func (sd *ServiceDiscovery) DiscoverDaemonSets() ([]*v1.DaemonSet, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allDaemonSets []*v1.DaemonSet
	for _, ns := range sd.config.Namespaces {
		daemonsets, err := sd.k8sClient.ListDaemonSets(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list daemonsets in namespace %s", ns)
			continue
		}
		allDaemonSets = append(allDaemonSets, daemonsets...)
	}

	return allDaemonSets, nil
}

// DiscoverReplicaSets discovers Kubernetes replica sets
func (sd *ServiceDiscovery) DiscoverReplicaSets() ([]*v1.ReplicaSet, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allReplicaSets []*v1.ReplicaSet
	for _, ns := range sd.config.Namespaces {
		replicaSets, err := sd.k8sClient.ListReplicaSets(ns, "")
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list replica sets in namespace %s", ns)
			continue
		}
		allReplicaSets = append(allReplicaSets, replicaSets...)
	}

	return allReplicaSets, nil
}

// DiscoverIngresses discovers Kubernetes ingresses
func (sd *ServiceDiscovery) DiscoverIngresses() ([]*networkingv1.Ingress, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	var allIngresses []*networkingv1.Ingress
	for _, ns := range sd.config.Namespaces {
		ingresses, err := sd.k8sClient.ListIngresses(ns)
		if err != nil {
			sd.logger.WithError(err).Errorf("Failed to list ingresses in namespace %s", ns)
			continue
		}
		allIngresses = append(allIngresses, ingresses...)
	}

	return allIngresses, nil
}

// DiscoverNamespaces discovers Kubernetes namespaces
func (sd *ServiceDiscovery) DiscoverNamespaces() ([]*v1.Namespace, error) {
	if sd.k8sClient == nil {
		return nil, ErrK8sNotConnected
	}

	namespaces, err := sd.k8sClient.ListNamespaces("")
	if err != nil {
		return nil, err
	}

	return namespaces, nil
}

// SearchServices searches for services by name or label
func (sd *ServiceDiscovery) SearchServices(query string) ([]*Service, error) {
	sd.mu.RLock()
	initialized := sd.initialized
	sd.mu.RUnlock()

	if !initialized {
		if err := sd.Initialize(); err != nil {
			return nil, err
		}
	}

	var results []*Service

	if sd.dockerClient != nil && sd.dockerClient.IsConnected() {
		containers, err := sd.dockerClient.ListContainers(true)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to list Docker containers")
		} else {
			for _, c := range containers {
				if matchesQuery(c.ID, query) || matchesQuery(c.Names[0], query) || matchesQuery(c.Image, query) {
					svc, err := sd.convertDockerContainerToService(&c)
					if err == nil {
						results = append(results, svc)
					}
				}
			}
		}
	}

	if sd.k8sClient != nil && sd.k8sClient.IsConnected() {
		for _, ns := range sd.config.Namespaces {
			// Search pods
			pods, _ := sd.k8sClient.ListPods(ns, "")
			for _, pod := range pods {
				if matchesQuery(pod.Name, query) || matchesQuery(pod.Namespace, query) {
					svc := &Service{
						Name:      pod.Name,
						Type:      "pod",
						Namespace: pod.Namespace,
						Labels:    pod.Labels,
						Labels:    pod.Labels,
					}
					results = append(results, svc)
				}
			}

			// Search services
			services, _ := sd.k8sClient.ListServices(ns, "")
			for _, svc := range services {
				if matchesQuery(svc.Name, query) || matchesQuery(svc.Namespace, query) {
					results = append(results, &Service{
						Name:      svc.Name,
						Type:      "service",
						Namespace: svc.Namespace,
						Labels:    svc.Labels,
					})
				}
			}
		}
	}

	return results, nil
}

// SearchServices searches for services by name or label
func (sd *ServiceDiscovery) SearchServices(query string) ([]*Service, error) {
	sd.mu.RLock()
	initialized := sd.initialized
	sd.mu.RUnlock()

	if !initialized {
		if err := sd.Initialize(); err != nil {
			return nil, err
		}
	}

	var results []*Service

	if sd.dockerClient != nil && sd.dockerClient.IsConnected() {
		containers, err := sd.dockerClient.ListContainers(true)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to list Docker containers")
		} else {
			for _, c := range containers {
				if matchesQuery(c.ID, query) || matchesQuery(c.Names[0], query) || matchesQuery(c.Image, query) {
					svc, err := sd.convertDockerContainerToService(&c)
					if err == nil {
						results = append(results, svc)
					}
				}
			}
		}
	}

	if sd.k8sClient != nil && sd.k8sClient.IsConnected() {
		for _, ns := range sd.config.Namespaces {
			// Search pods
			pods, _ := sd.k8sClient.ListPods(ns, "")
			for _, pod := range pods {
				if matchesQuery(pod.Name, query) || matchesQuery(pod.Namespace, query) {
					svc := &Service{
						Name:      pod.Name,
						Type:      "pod",
						Namespace: pod.Namespace,
						Labels:    pod.Labels,
					}
					results = append(results, svc)
				}
			}

			// Search services
			services, _ := sd.k8sClient.ListServices(ns, "")
			for _, svc := range services {
				if matchesQuery(svc.Name, query) || matchesQuery(svc.Namespace, query) {
					results = append(results, &Service{
						Name:      svc.Name,
						Type:      "service",
						Namespace: svc.Namespace,
						Labels:    svc.Labels,
					})
				}
			}
		}
	}

	return results, nil
}

// matchesQuery checks if a query matches a string
func matchesQuery(value, query string) bool {
	return value != "" && containsIgnoreCase(value, query)
}

// containsIgnoreCase checks if a string contains a substring (case-insensitive)
func containsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// GetServiceHealthCheck returns the health check configuration for a service
func (sd *ServiceDiscovery) GetServiceHealthCheck(serviceName string) (ServiceHealthConfig, error) {
	return ServiceHealthConfig{
		Endpoint:       fmt.Sprintf("/health"),
		Interval:       10 * time.Second,
		Timeout:        5 * time.Second,
		HealthyThreshold: 2,
		UnhealthyThreshold: 3,
	}, nil
}

// SubscribeToEvents subscribes to service discovery events
func (sd *ServiceDiscovery) SubscribeToEvents(eventType EventType, handler func(*Event)) error {
	if sd.eventBus == nil {
		sd.eventBus = NewEventBus()
	}

	sd.eventBus.Subscribe(eventType, &Subscriber{
		ID:         fmt.Sprintf("service-discovery-%d", time.Now().UnixNano()),
		HandleFunc: handler,
	})

	return nil
}

// UnsubscribeFromEvents unsubscribes from service discovery events
func (sd *ServiceDiscovery) UnsubscribeFromEvents(eventType EventType, handler func(*Event)) error {
	if sd.eventBus == nil {
		return ErrEventBusNotInitialized
	}

	sub := &Subscriber{
		ID:         fmt.Sprintf("service-discovery-%d", time.Now().UnixNano()),
		HandleFunc: handler,
	}

	sd.eventBus.Unsubscribe(eventType, sub)
	return nil
}

// StartHotReload starts hot reloading service discoveries
func (sd *ServiceDiscovery) StartHotReload() {
	sd.mu.Lock()
	sd.hotReload = true
	sd.mu.Unlock()

	if sd.dockerClient != nil {
		sd.dockerClient.EnableHotReload()
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-sd.ctx.Done():
				return
			case <-ticker.C:
				if sd.hotReload {
					sd.DiscoverServices()
				}
			}
		}
	}()
}

// StopHotReload stops hot reloading
func (sd *ServiceDiscovery) StopHotReload() {
	sd.mu.Lock()
	sd.hotReload = false
	sd.mu.Unlock()

	if sd.dockerClient != nil {
		sd.dockerClient.DisableHotReload()
	}
}

// GetLastDiscoveryTime returns the last discovery time
func (sd *ServiceDiscovery) GetLastDiscoveryTime() time.Time {
	sd.mu.RLock()
	defer sd.mu.RUnlock()
	return sd.lastDiscovery
}

// GetDiscoveryStats returns statistics about service discovery
func (sd *ServiceDiscovery) GetDiscoveryStats() map[string]interface{} {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	stats := map[string]interface{}{
		"initialized": sd.initialized,
		"hot_reload":  sd.hotReload,
		"last_discovery": sd.lastDiscovery,
	}

	if sd.dockerClient != nil {
		stats["docker_client"] = map[string]interface{}{
			"connected": sd.dockerClient.IsConnected(),
		}
	}

	if sd.k8sClient != nil {
		stats["kubernetes_client"] = map[string]interface{}{
			"connected": sd.k8sClient.IsConnected(),
		}
	}

	if sd.eventBus != nil {
		stats["event_bus_subscribers"] = map[string]interface{}{
			"total": sd.eventBus.GetAllSubscriberCount(),
		}
	}

	if sd.eventStream != nil {
		stats["redis_stream"] = map[string]interface{}{
			"connected":    sd.eventStream.IsConnected(),
			"stream_length": sd.eventStream.StreamLength(),
		}
	}

	if sd.hub != nil {
		stats["websocket_connections"] = sd.hub.GetConnectionCount()
	}

	return stats
}

// Shutdown shuts down the service discovery
func (sd *ServiceDiscovery) Shutdown() {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	if sd.cancel != nil {
		sd.cancel()
	}

	if sd.dockerClient != nil {
		sd.dockerClient.Shutdown()
	}

	if sd.k8sClient != nil {
		sd.k8sClient.Shutdown()
	}

	if sd.eventBus != nil {
		sd.eventBus.Shutdown()
	}

	if sd.eventStream != nil {
		sd.eventStream.Shutdown()
	}

	if sd.hub != nil {
		sd.hub.Stop()
	}

	sd.initialized = false

	sd.logger.Info("Service Discovery shut down")
}

// handleStreamEvent handles events from Redis stream
func (sd *ServiceDiscovery) handleStreamEvent(event *Event) {
	sd.logger.WithFields(logrus.Fields{
		"type":       event.Type,
		"timestamp":  event.Timestamp,
		"metadata":   event.Metadata,
	}).Info("Received event from Redis stream")

	// Broadcast to WebSocket clients
	if sd.hub != nil {
		sd.hub.BroadcastEvent(event)
	}

	// Publish to event bus
	if sd.eventBus != nil {
		sd.eventBus.Publish(event)
	}
}

// convertDockerContainerToService converts a Docker container to a Service
func (sd *ServiceDiscovery) convertDockerContainerToService(container *types.Container) (*Service, error) {
	svc := &Service{
		ID:           container.ID,
		Name:         container.Names[0],
		Type:         "docker",
		Status:       "unknown",
		HealthStatus: "unknown",
		Image:        container.Image,
		Ports:        make(map[string]string),
		Labels:       make(map[string]string),
		Created:      time.Unix(container.Created, 0),
		LastUpdated:  time.Now(),
	}

	// Extract labels
	for k, v := range container.Labels {
		svc.Labels[k] = v
	}

	// Extract ports
	for port, bindings := range container.Ports {
		if bindings != nil {
			for _, binding := range bindings {
				key := port.String()
				svc.Ports[key] = fmt.Sprintf("%s:%s", binding.HostIP, binding.HostPort)
			}
		}
	}

	// Set status based on container state
	switch container.State {
	case "running":
		svc.Status = "running"
		svc.HealthStatus = "healthy"
	case "exited":
		svc.Status = "stopped"
		svc.HealthStatus = "unhealthy"
	case "restarting":
		svc.Status = "restarting"
		svc.HealthStatus = "unknown"
	case "paused":
		svc.Status = "paused"
		svc.HealthStatus = "unknown"
	case "dead":
		svc.Status = "dead"
		svc.HealthStatus = "unhealthy"
	default:
		svc.Status = "unknown"
		svc.HealthStatus = "unknown"
	}

	return svc, nil
}

// convertDockerContainerToContainer converts a Docker container to a Container struct
func (sd *ServiceDiscovery) convertDockerContainerToContainer(container *types.Container) (*Container, error) {
	c := &Container{
		ID:           container.ID,
		Name:         container.Names[0],
		Image:        container.Image,
		Status:       ServiceStatus(container.State),
		State:        container.State,
		Created:      time.Unix(container.Created, 0),
		Command:      container.Command,
		Labels:       container.Labels,
		Ports:        make([]Port, 0),
		Mounts:       make([]Mount, 0),
		Networks:     make([]string, 0),
		HealthStatus: "unknown",
	}

	// Convert ports
	for port, bindings := range container.Ports {
		if bindings != nil {
			for _, binding := range bindings {
				c.Ports = append(c.Ports, Port{
					ContainerPort: int(port.Int()),
					HostPort:      int(binding.HostPort),
					Protocol:      port.Proto(),
					HostIP:        binding.HostIP,
					IP:            binding.HostIP,
					PrivatePort:   int(port.Int()),
					PublicPort:    int(binding.HostPort),
					Type:          port.Proto(),
				})
			}
		}
	}

	// Convert networks
	if container.NetworkSettings != nil {
		for netName := range container.NetworkSettings.Networks {
			c.Networks = append(c.Networks, netName)
		}
	}

	return c, nil
}

// convertK8sResourceToService converts a Kubernetes resource to a Service
func (sd *ServiceDiscovery) convertK8sResourceToService(obj interface{}) (*Service, error) {
	svc := &Service{
		Type:       "kubernetes",
		Status:     "unknown",
		Labels:     make(map[string]string),
		Created:    time.Now(),
		LastUpdated: time.Now(),
	}

	switch v := obj.(type) {
	case *v1.Pod:
		svc.Name = v.Name
		svc.Namespace = v.Namespace
		svc.Labels = v.Labels
		svc.Status = string(v.Status.Phase)
	case *v1.Service:
		svc.Name = v.Name
		svc.Namespace = v.Namespace
		svc.Labels = v.Labels
		svc.Status = "running"
	case *v1.Deployment:
		svc.Name = v.Name
		svc.Namespace = v.Namespace
		svc.Labels = v.Labels
		svc.Status = "running"
	case *v1.ConfigMap:
		svc.Name = v.Name
		svc.Namespace = v.Namespace
		svc.Labels = v.Labels
		svc.Status = "active"
	case *v1.Secret:
		svc.Name = v.Name
		svc.Namespace = v.Namespace
		svc.Labels = v.Labels
		svc.Status = "active"
	default:
		return nil, fmt.Errorf("unsupported Kubernetes resource type: %T", obj)
	}

	return svc, nil
}

// GetServices returns all discovered services
func (sd *ServiceDiscovery) GetServices() ([]*Service, error) {
	if err := sd.Initialize(); err != nil {
		return nil, err
	}

	var services []*Service

	// Discover Docker services
	if sd.dockerClient != nil && sd.dockerClient.IsConnected() {
		containers, err := sd.dockerClient.ListContainers(true)
		if err != nil {
			sd.logger.WithError(err).Error("Failed to list Docker containers")
		} else {
			for _, c := range containers {
				svc, err := sd.convertDockerContainerToService(&c)
				if err != nil {
					sd.logger.WithError(err).Error("Failed to convert container to service")
					continue
				}
				services = append(services, svc)
			}
		}
	}

	// Discover Kubernetes services
	if sd.k8sClient != nil && sd.k8sClient.IsConnected() {
		// Get all resource types
		for _, ns := range sd.config.Namespaces {
			for _, rt := range sd.config.ResourceTypes {
				resources, err := sd.collectK8sResources(rt, ns)
				if err != nil {
					sd.logger.WithError(err).Errorf("Failed to collect %s resources", rt)
					continue
				}

				for _, obj := range resources {
					svc, err := sd.convertK8sResourceToService(obj)
					if err != nil {
						sd.logger.WithError(err).Error("Failed to convert resource to service")
						continue
					}
					services = append(services, svc)
				}
			}
		}
	}

	return services, nil
}

// collectK8sResources collects resources of a specific type from Kubernetes
func (sd *ServiceDiscovery) collectK8sResources(resourceType, namespace string) ([]interface{}, error) {
	var resources []interface{}

	switch resourceType {
	case "pod":
		pods, err := sd.k8sClient.ListPods(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pod := range pods {
			resources = append(resources, pod)
		}
	case "service":
		services, err := sd.k8sClient.ListServices(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, svc := range services {
			resources = append(resources, svc)
		}
	case "deployment":
		deployments, err := sd.k8sClient.ListDeployments(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, dep := range deployments {
			resources = append(resources, dep)
		}
	case "statefulset":
		statefulsets, err := sd.k8sClient.ListStatefulSets(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, ss := range statefulsets {
			resources = append(resources, ss)
		}
	case "daemonset":
		daemonsets, err := sd.k8sClient.ListDaemonSets(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, ds := range daemonsets {
			resources = append(resources, ds)
		}
	case "replicaset":
		replicaSets, err := sd.k8sClient.ListReplicaSets(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, rs := range replicaSets {
			resources = append(resources, rs)
		}
	case "configmap":
		configmaps, err := sd.k8sClient.ListConfigMaps(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, cm := range configmaps {
			resources = append(resources, cm)
		}
	case "secret":
		secrets, err := sd.k8sClient.ListSecrets(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, secret := range secrets {
			resources = append(resources, secret)
		}
	case "ingress":
		ingresses, err := sd.k8sClient.ListIngresses(namespace)
		if err != nil {
			return nil, err
		}
		for _, ing := range ingresses {
			resources = append(resources, &ing)
		}
	case "persistentvolumeclaim":
		pvcs, err := sd.k8sClient.ListPersistentVolumeClaims(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pvc := range pvcs {
			resources = append(resources, pvc)
		}
	case "namespace":
		namespaces, err := sd.k8sClient.ListNamespaces("")
		if err != nil {
			return nil, err
		}
		for _, ns := range namespaces {
			resources = append(resources, ns)
		}
	case "poddisruptionbudget":
		pdbs, err := sd.k8sClient.ListPodDisruptionBudgets(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pdb := range pdbs {
			resources = append(resources, pdb)
		}
	case "horizontalpodautoscaler":
		hpas, err := sd.k8sClient.ListHorizontalPodAutoscalers(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, hpa := range hpas {
			resources = append(resources, hpa)
		}
	case "cronjob":
		cronjobs, err := sd.k8sClient.ListCronJobs(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, cj := range cronjobs {
			resources = append(resources, cj)
		}
	case "job":
		jobs, err := sd.k8sClient.ListJobs(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, j := range jobs {
			resources = append(resources, j)
		}
	case "storageclass":
		storageClasses, err := sd.k8sClient.ListStorageClasses()
		if err != nil {
			return nil, err
		}
		for _, sc := range storageClasses {
			resources = append(resources, sc)
		}
	case "persistentvolume":
		pvs, err := sd.k8sClient.ListPersistentVolumes("")
		if err != nil {
			return nil, err
		}
		for _, pv := range pvs {
			resources = append(resources, pv)
		}
	case "node":
		nodes, err := sd.k8sClient.ListNodes("")
		if err != nil {
			return nil, err
		}
		for _, node := range nodes {
			resources = append(resources, node)
		}
	case "resourcequota":
		resourceQuotas, err := sd.k8sClient.ListResourceQuotas(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, rq := range resourceQuotas {
			resources = append(resources, rq)
		}
	case "limitrange":
		limitRanges, err := sd.k8sClient.ListLimitRanges(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, lr := range limitRanges {
			resources = append(resources, lr)
		}
	case "podtemplate":
		podTemplates, err := sd.k8sClient.ListPodTemplates(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pt := range podTemplates {
			resources = append(resources, pt)
		}
	case "replicationcontroller":
		controllers, err := sd.k8sClient.ListReplicationControllers(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, rc := range controllers {
			resources = append(resources, rc)
		}
	case "endpoint":
		endpoints, err := sd.k8sClient.ListEndpoints(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, ep := range endpoints {
			resources = append(resources, ep)
		}
	case "event":
		events, err := sd.k8sClient.ListEvents(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, evt := range events {
			resources = append(resources, evt)
		}
	case "serviceaccount":
		serviceAccounts, err := sd.k8sClient.ListServiceAccounts(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, sa := range serviceAccounts {
			resources = append(resources, sa)
		}
	case "role":
		roles, err := sd.k8sClient.ListRoles(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, role := range roles {
			resources = append(resources, role)
		}
	case "clusterrole":
		clusterRoles, err := sd.k8sClient.ListClusterRoles("")
		if err != nil {
			return nil, err
		}
		for _, cr := range clusterRoles {
			resources = append(resources, cr)
		}
	case "rolebinding":
		roleBindings, err := sd.k8sClient.ListRoleBindings(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, rb := range roleBindings {
			resources = append(resources, rb)
		}
	case "clusterrolebinding":
		clusterRoleBindings, err := sd.k8sClient.ListClusterRoleBindings("")
		if err != nil {
			return nil, err
		}
		for _, crb := range clusterRoleBindings {
			resources = append(resources, crb)
		}
	case "priorityclass":
		priorityClasses, err := sd.k8sClient.ListPriorityClasses()
		if err != nil {
			return nil, err
		}
		for _, pc := range priorityClasses {
			resources = append(resources, pc)
		}
	case "podsecuritypolicy":
		podSecurityPolicies, err := sd.k8sClient.ListPodSecurityPolicies()
		if err != nil {
			return nil, err
		}
		for _, psp := range podSecurityPolicies {
			resources = append(resources, psp)
		}
	case "networkpolicy":
		networkPolicies, err := sd.k8sClient.ListNetworkPolicies(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, np := range networkPolicies {
			resources = append(resources, np)
		}
	case "mutatingwebhookconfiguration":
		mutatingWebhooks, err := sd.k8sClient.ListMutatingWebhookConfigurations()
		if err != nil {
			return nil, err
		}
		for _, mwc := range mutatingWebhooks {
			resources = append(resources, mwc)
		}
	case "validatingwebhookconfiguration":
		validatingWebhooks, err := sd.k8sClient.ListValidatingWebhookConfigurations()
		if err != nil {
			return nil, err
		}
		for _, vwc := range validatingWebhooks {
			resources = append(resources, vwc)
		}
	case "podmonitor":
		podMonitors, err := sd.k8sClient.ListPodMonitors(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pm := range podMonitors {
			resources = append(resources, pm)
		}
	case "servicemonitor":
		serviceMonitors, err := sd.k8sClient.ListServiceMonitors(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, sm := range serviceMonitors {
			resources = append(resources, sm)
		}
	case "prometheusrule":
		prometheusRules, err := sd.k8sClient.ListPrometheusRules(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, pr := range prometheusRules {
			resources = append(resources, pr)
		}
	case "alertmanager":
		alertManagers, err := sd.k8sClient.ListAlertManagers(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, am := range alertManagers {
			resources = append(resources, am)
		}
	case "thanosruler":
		thanosRulers, err := sd.k8sClient.ListThanosRulers(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, tr := range thanosRulers {
			resources = append(resources, tr)
		}
	case "prometheus":
		prometheuses, err := sd.k8sClient.ListPrometheuses(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, p := range prometheuses {
			resources = append(resources, p)
		}
	case "grafanadatasource":
		grafanaDataSources, err := sd.k8sClient.ListGrafanaDataSources(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, gd := range grafanaDataSources {
			resources = append(resources, gd)
		}
	case "grafanadashboard":
		grafanaDashboards, err := sd.k8sClient.ListGrafanaDashboards(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, gd := range grafanaDashboards {
			resources = append(resources, gd)
		}
	case "grafanadashboardprovider":
		grafanaDashboardProviders, err := sd.k8sClient.ListGrafanaDashboardProviders(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, gd := range grafanaDashboardProviders {
			resources = append(resources, gd)
		}
	case "thanosruler":
		thanosRulers, err := sd.k8sClient.ListThanosRulers(namespace, "")
		if err != nil {
			return nil, err
		}
		for _, tr := range thanosRulers {
			resources = append(resources, tr)
		}
	default:
		return nil, fmt.Errorf("unsupported resource type: %s", resourceType)
	}

	return resources, nil
}
