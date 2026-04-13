package catalog

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceDiscoveryEngine implements the ServiceDiscovery interface
type ServiceDiscoveryEngine struct {
	mu               sync.RWMutex
	config           *DiscoveryConfig
	metricsCollector MetricsCollector
	cache            ResourceCache
	eventBus         EventBus
	handlers         []DiscoveryEventHandler
	logger           *logrus.Logger
	healthWatcher    ContainerHealthWatcher
	resourceWatcher  ResourceWatcher
	statusTracker    ResourceStatusTracker
	discoveryMetrics DiscoveryMetrics
	lastDiscovery    time.Time
	muDiscovery      sync.Mutex
}

// NewServiceDiscoveryEngine creates a new service discovery engine
func NewServiceDiscoveryEngine(cfg *DiscoveryConfig) *ServiceDiscoveryEngine {
	if cfg == nil {
		cfg = &DiscoveryConfig{
			EnableDocker:       true,
			EnableKubernetes:   true,
			RefreshInterval:    60 * time.Second,
			HealthCheckTimeout: 10 * time.Second,
			DockerSocket:       "unix:///var/run/docker.sock",
			KubeconfigPath:     "",
			Namespaces:         []string{"default"},
			ResourceTypes:      []string{ResourceTypePod, ResourceTypeDeployment, ResourceTypeService},
			IncludeMetrics:     true,
			EnableEvents:       true,
		}
	}

	engine := &ServiceDiscoveryEngine{
		config:     cfg,
		logger:     logrus.New(),
		cache:      NewResourceCache(),
	}

	engine.logger.SetLevel(logrus.InfoLevel)
	engine.logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Create metrics collector
	engine.metricsCollector = NewMetricsCollector(cfg)

	// Create resource watcher if enabled
	if cfg.EnableKubernetes {
		engine.resourceWatcher = NewK8sResourceWatcher(cfg)
	}

	// Create health watcher if enabled
	if cfg.EnableDocker {
		engine.healthWatcher = NewContainerHealthWatcher(cfg)
	}

	// Create event bus if enabled
	if cfg.EnableEvents {
		eventBus := NewEventBus()
		engine.eventBus = eventBus
		engine.initEventHandlers(eventBus)
	}

	return engine
}

// SetLogger sets the logger for the engine
func (e *ServiceDiscoveryEngine) SetLogger(logger *logrus.Logger) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.logger = logger
}

// SetMetricsCollector sets the metrics collector
func (e *ServiceDiscoveryEngine) SetMetricsCollector(collector MetricsCollector) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.metricsCollector = collector
}

// SetCache sets the resource cache
func (e *ServiceDiscoveryEngine) SetCache(cache ResourceCache) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cache = cache
}

// SetEventBus sets the event bus
func (e *ServiceDiscoveryEngine) SetEventBus(bus EventBus) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.eventBus = bus
}

// AddHandler adds a discovery event handler
func (e *ServiceDiscoveryEngine) AddHandler(handler DiscoveryEventHandler) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.handlers = append(e.handlers, handler)
}

// GetConfig returns the current configuration
func (e *ServiceDiscoveryEngine) GetConfig() *DiscoveryConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.config
}

// SetConfig updates the configuration
func (e *ServiceDiscoveryEngine) SetConfig(cfg *DiscoveryConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.config = cfg
}

// Discovery runs the service discovery operation
func (e *ServiceDiscoveryEngine) Discovery() (DiscoveryResult, error) {
	return e.Discover()
}

// Discover runs the service discovery operation
func (e *ServiceDiscoveryEngine) Discover() (DiscoveryResult, result) {
	e.muDiscovery.Lock()
	startTime := time.Now()
	e.muDiscovery.Unlock()

	e.logger.Info("Starting service discovery")

	var result DiscoveryResult
	var errors []string
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Configure metrics collector
	e.metricsCollector.SetConfig(e.config)

	// Discover containers if Docker is enabled
	if e.config.EnableDocker {
		wg.Add(1)
		go func() {
			defer wg.Done()
			containers, err := e.metricsCollector.CollectContainers()
			if err != nil {
				e.logger.WithError(err).Error("Failed to collect containers")
				mu.Lock()
				errors = append(errors, "Container discovery failed: "+err.Error())
				mu.Unlock()
				return
			}

			// Process containers and cache them
			for _, container := range containers {
				e.cache.Set(container.ID, container)

				// Check if status changed
				oldContainer, exists := e.cache.Get(container.ID)
				if exists {
					if oldContainer.Status != container.Status {
						e.logger.WithFields(logrus.Fields{
							"container": container.Name,
							"old":      oldContainer.Status,
							"new":      container.Status,
						}).Info("Container status changed")

						if e.eventBus != nil {
							eventType := EventContainerStatusChanged
							if container.Status == StatusStopped {
								eventType = EventContainerStopped
							} else if container.Status == StatusRunning {
								eventType = EventContainerStarted
							}

							e.eventBus.Publish(&Event{
								Type:      eventType,
								Timestamp: time.Now(),
								Container: container,
							})
						}
					}
				}
			}

			// Update discovery metrics
			e.mu.Lock()
			e.discoveryMetrics.ContainersFound += len(containers)
			e.mu.Unlock()

			mu.Lock()
			result.Containers = containers
			mu.Unlock()
		}()
	}

	// Discover images if Docker is enabled
	if e.config.EnableDocker {
		wg.Add(1)
		go func() {
			defer wg.Done()
			images, err := e.metricsCollector.CollectImages()
			if err != nil {
				e.logger.WithError(err).Error("Failed to collect images")
				mu.Lock()
				errors = append(errors, "Image discovery failed: "+err.Error())
				mu.Unlock()
				return
			}

			// Process images and cache them
			for _, image := range images {
				digest := image.Digest
				if digest == "" && len(image.RepoTags) > 0 {
					digest = image.RepoTags[0]
				}
				if digest != "" {
					e.cache.SetImage(digest, image)
				}
			}

			// Update discovery metrics
			e.mu.Lock()
			e.discoveryMetrics.ImagesFound += len(images)
			e.mu.Unlock()

			mu.Lock()
			result.Images = images
			mu.Unlock()
		}()
	}

	// Discover networks if Docker is enabled
	if e.config.EnableDocker {
		wg.Add(1)
		go func() {
			defer wg.Done()
			networks, err := e.metricsCollector.CollectNetworks()
			if err != nil {
				e.logger.WithError(err).Error("Failed to collect networks")
				mu.Lock()
				errors = append(errors, "Network discovery failed: "+err.Error())
				mu.Unlock()
				return
			}

			// Process networks and cache them
			for _, network := range networks {
				e.cache.SetNetwork(network.ID, network)
			}

			// Update discovery metrics
			e.mu.Lock()
			e.discoveryMetrics.NetworksFound += len(networks)
			e.mu.Unlock()

			mu.Lock()
			result.Networks = networks
			mu.Unlock()
		}()
	}

	// Discover Kubernetes resources if enabled
	if e.config.EnableKubernetes {
		wg.Add(1)
		go func() {
			defer wg.Done()
			k8sResources, err := e.metricsCollector.CollectKubernetesResources()
			if err != nil {
				e.logger.WithError(err).Error("Failed to collect Kubernetes resources")
				mu.Lock()
				errors = append(errors, "Kubernetes discovery failed: "+err.Error())
				mu.Unlock()
				return
			}

			// Process K8s resources and cache them
			for _, resource := range k8sResources {
				e.cache.SetK8sResource(string(resource.Type), resource.Namespace, resource.Name, resource)

				// Update discovery metrics
				e.mu.Lock()
				e.discoveryMetrics.K8sResourcesFound += 1
				e.mu.Unlock()
			}

			mu.Lock()
			result.K8sResources = k8sResources
			mu.Unlock()
		}()
	}

	// Wait for all discovery operations to complete
	wg.Wait()

	// Calculate duration
	duration := time.Since(startTime)

	// Handle errors
	if len(errors) > 0 {
		e.logger.WithFields(logrus.Fields{
			"errors": len(errors),
			"duration": duration,
		}).Warn("Discovery completed with errors")
		result.Errors = errors
	}

	// Update discovery metrics
	e.mu.Lock()
	e.discoveryMetrics.Duration = int64(duration / time.Millisecond)
	e.lastDiscovery = time.Now()
	e.mu.Unlock()

	// Publish completion event
	if e.eventBus != nil && len(result.Errors) == 0 {
		e.eventBus.Publish(&Event{
			Type:      EventDiscoveryCompleted,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"containers_found": len(result.Containers),
				"images_found":   len(result.Images),
				"networks_found": len(result.Networks),
				"k8s_resources_found": len(result.K8sResources),
			},
		})
	} else if e.eventBus != nil {
		e.eventBus.Publish(&Event{
			Type:      EventDiscoveryFailed,
			Timestamp: time.Now(),
			Error:     errorsToError(errors),
			Metadata: map[string]interface{}{
				"error_count": len(result.Errors),
			},
		})
	}

	result.Timestamp = time.Now()
	result.Success = len(result.Errors) == 0

	return result, nil
}

// CollectMetrics collects current metrics
func (e *ServiceDiscoveryEngine) CollectMetrics() MetricsCollector {
	return e.metricsCollector
}

// GetServiceStatus returns the status of a service by its ID
func (e *ServiceDiscoveryEngine) GetServiceStatus(serviceID string) (ServiceStatus, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.statusTracker == nil {
		return StatusUnknown, nil
	}

	return e.statusTracker.GetStatus("container", serviceID)
}

// HealthCheck performs a health check on a service
func (e *ServiceDiscoveryEngine) HealthCheck(serviceID string) error {
	e.mu.RLock()
	cache := e.cache
	metricsCollector := e.metricsCollector
	e.mu.RUnlock()

	// Get the container
	container, exists := cache.Get(serviceID)
	if !exists {
		return ErrServiceNotFound
	}

	// Check health status
	if container.Health == nil {
		return nil // No health check configured
	}

	if container.Health.Status == "healthy" || container.Health.Status == "starting" {
		return nil
	}

	e.logger.WithFields(logrus.Fields{
		"container": container.Name,
		"status":    container.Health.Status,
	}).Warn("Container health check failed")

	return nil
}

// UpdateStatus updates the status of a service
func (e *ServiceDiscoveryEngine) UpdateStatus(serviceID string, status ServiceStatus) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.statusTracker == nil {
		e.statusTracker = NewResourceStatusTracker()
	}

	return e.statusTracker.TrackStatus("container", serviceID, status)
}

// Run starts the service discovery with periodic refresh
func (e *ServiceDiscoveryEngine) Run(ctx <-chan struct{}) {
	interval := e.config.RefreshInterval
	if interval <= 0 {
		interval = 60 * time.Second
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Initial discovery
	if _, err := e.Discover(); err != nil {
		e.logger.WithError(err).Error("Initial discovery failed")
	}

	for {
		select {
		case <-ctx:
			e.logger.Info("Stopping service discovery")
			return
		case <-ticker.C:
			e.logger.Debug("Running scheduled service discovery")

			if _, err := e.Discover(); err != nil {
				e.logger.WithError(err).Error("Scheduled discovery failed")
			}
		}
	}
}

// GetMetrics returns the current discovery metrics
func (e *ServiceDiscoveryEngine) GetMetrics() *DiscoveryMetrics {
	e.mu.Lock()
	defer e.mu.Unlock()

	return &e.discoveryMetrics
}

// GetCacheStats returns cache statistics
func (e *ServiceDiscoveryEngine) GetCacheStats() *CacheMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.cache.GetStats()
}

// GetContainers returns all discovered containers
func (e *ServiceDiscoveryEngine) GetContainers() []*Container {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cache.AllContainers()
}

// GetImages returns all discovered images
func (e *ServiceDiscoveryEngine) GetImages() []*Image {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cache.AllImages()
}

// GetNetworks returns all discovered networks
func (e *ServiceDiscoveryEngine) GetNetworks() []*Network {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cache.AllNetworks()
}

// GetK8sResources returns all discovered Kubernetes resources
func (e *ServiceDiscoveryEngine) GetK8sResources() []*ServiceResource {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.cache.AllK8sResources()
}

// GetK8sResource returns a specific Kubernetes resource
func (e *ServiceDiscoveryEngine) GetK8sResource(resourceType, namespace, name string) (*ServiceResource, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	resource, exists := e.cache.GetK8sResource(resourceType, namespace, name)
	if !exists {
		return nil, ErrServiceNotFound
	}

	return resource, nil
}

// GetCachedContainer returns a cached container
func (e *ServiceDiscoveryEngine) GetCachedContainer(containerID string) (*Container, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	container, exists := e.cache.Get(containerID)
	if !exists {
		return nil, ErrServiceNotFound
	}

	return container, nil
}

// GetCachedImage returns a cached image
func (e *ServiceDiscoveryEngine) GetCachedImage(digest string) (*Image, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	image, exists := e.cache.GetImage(digest)
	if !exists {
		return nil, ErrServiceNotFound
	}

	return image, nil
}

// GetCachedNetwork returns a cached network
func (e *ServiceDiscoveryEngine) GetCachedNetwork(networkID string) (*Network, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	network, exists := e.cache.GetNetwork(networkID)
	if !exists {
		return nil, ErrServiceNotFound
	}

	return network, nil
}

// SetMetricsCollector sets the metrics collector for this engine
func (e *ServiceDiscoveryEngine) SetMetricsCollector(mc MetricsCollector) {
	e.metricsCollector = mc
}

// SetResourceWatcher sets the Kubernetes resource watcher
func (e *ServiceDiscoveryEngine) SetResourceWatcher(watcher ResourceWatcher) {
	e.resourceWatcher = watcher
}

// SetHealthWatcher sets the container health watcher
func (e *ServiceDiscoveryEngine) SetHealthWatcher(watcher ContainerHealthWatcher) {
	e.healthWatcher = watcher
}

// GetHandlers returns all registered discovery event handlers
func (e *ServiceDiscoveryEngine) GetHandlers() []DiscoveryEventHandler {
	e.mu.RLock()
	defer e.mu.RUnlock()

	handlers := make([]DiscoveryEventHandler, len(e.handlers))
	copy(handlers, e.handlers)
	return handlers
}

// initEventHandlers initializes event handlers for the event bus
func (e *ServiceDiscoveryEngine) initEventHandlers(bus EventBus) {
	// Register container status change handler
	bus.Subscribe(EventContainerStarted, &containerEventHandler{
		engine:    e,
		onStart: func(c *Container) {
			e.logger.WithFields(logrus.Fields{
				"container": c.Name,
				"id":        c.ID,
			}).Info("Container started")
		},
		onStop: func(c *Container) {
			e.logger.WithFields(logrus.Fields{
				"container": c.Name,
				"id":        c.ID,
			}).Info("Container stopped")
		},
	})

	// Register health status change handler
	bus.Subscribe(EventContainerHealthChanged, &containerEventHandler{
		engine: e,
		onHealth: func(c *Container, oldStatus, newStatus string) {
			e.logger.WithFields(logrus.Fields{
				"container": c.Name,
				"id":        c.ID,
				"old":       oldStatus,
				"new":       newStatus,
			}).Warn("Container health status changed")
		},
	})
}

// containerEventHandler handles container events
type containerEventHandler struct {
	engine   *ServiceDiscoveryEngine
	onStart  func(*Container)
	onStop   func(*Container)
	onHealth func(*Container, string, string)
}

// HandleEvent handles events from the event bus
func (h *containerEventHandler) HandleEvent(event *Event) {
	switch event.Type {
	case EventContainerStarted:
		if event.Container != nil && h.onStart != nil {
			h.onStart(event.Container)
		}
	case EventContainerStopped:
		if event.Container != nil && h.stop != nil {
			h.stop(event.Container)
		}
	case EventContainerHealthChanged:
		if event.Container != nil && h.onHealth != nil {
			h.onHealth(event.Container, "unknown", "unknown")
		}
	}
}
