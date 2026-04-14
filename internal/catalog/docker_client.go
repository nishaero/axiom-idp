package catalog

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
)

// DockerClient provides integration with the Docker daemon for service discovery
type DockerClient struct {
	mu          sync.RWMutex
	client      *client.Client
	config      *DiscoveryConfig
	logger      *logrus.Logger
	ctx         context.Context
	cancel      context.CancelFunc
	eventChan   chan *Event
	hotReload   bool
	initialized bool
}

// NewDockerClient creates a new Docker client instance
func NewDockerClient(cfg *DiscoveryConfig, logger *logrus.Logger) *DockerClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &DockerClient{
		config:      cfg,
		logger:      logger,
		ctx:         ctx,
		cancel:      cancel,
		eventChan:   make(chan *Event, 100),
		hotReload:   false,
		initialized: false,
	}
}

// SetConfig updates the configuration
func (dc *DockerClient) SetConfig(cfg *DiscoveryConfig) {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.config = cfg
}

// GetConfig returns the current configuration
func (dc *DockerClient) GetConfig() *DiscoveryConfig {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.config
}

// initializeClient initializes the Docker client connection
func (dc *DockerClient) initializeClient() error {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.initialized && dc.client != nil {
		return nil
	}

	// Use default socket or configured socket
	socket := dc.config.DockerSocket
	if socket == "" {
		socket = "unix:///var/run/docker.sock"
	}

	// Extract socket path
	var socketPath string
	if strings.HasPrefix(socket, "unix://") {
		socketPath = socket[7:]
	} else if strings.HasPrefix(socket, "tcp://") {
		// Handle TCP connection for remote Docker
		socketPath = strings.TrimPrefix(socket, "tcp://")
	} else {
		socketPath = socket
	}

	// Create Docker client with options
	opts := []client.Opt{
		client.WithAPIVersionNegotiation(),
		client.WithTimeout(30 * time.Second),
	}

	// Determine protocol and add appropriate client option
	if strings.HasPrefix(socket, "tcp://") {
		opts = append(opts, client.WithHost(socket))
	} else {
		opts = append(opts, client.WithHost(fmt.Sprintf("unix://%s", socketPath)))
	}

	cli, err := client.NewClientWithOpts(opts...)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Verify connection to Docker daemon
	if err := cli.Ping(dc.ctx); err != nil {
		return fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	dc.client = cli
	dc.initialized = true

	dc.logger.WithFields(logrus.Fields{
		"socket": socket,
		"protocol": strings.Contains(socket, "tcp") ? "tcp" : "unix",
	}).Info("Docker client initialized and connected")

	return nil
}

// Ping checks the Docker daemon connection
func (dc *DockerClient) Ping() error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotInitialized
	}

	_, err := client.Ping(dc.ctx)
	return err
}

// IsConnected returns true if the client is connected to Docker
func (dc *DockerClient) IsConnected() bool {
	return dc.client != nil && dc.initialized
}

// ListContainers retrieves all containers from Docker
func (dc *DockerClient) ListContainers(all bool) ([]types.Container, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	opts := container.ListOptions{
		All: all,
	}

	containers, err := client.ContainerList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	return containers, nil
}

// GetContainerDetails retrieves detailed information about a specific container
func (dc *DockerClient) GetContainerDetails(containerID string) (*types.ContainerJSON, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	inspect, err := client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}

	return &inspect, nil
}

// GetContainerStats retrieves real-time statistics for a container
func (dc *DockerClient) GetContainerStats(containerID string, stream bool) (container.StatsResponse, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return container.StatsResponse{}, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	stats, err := client.ContainerStats(ctx, containerID, stream)
	if err != nil {
		return container.StatsResponse{}, fmt.Errorf("failed to get stats for container %s: %w", containerID, err)
	}

	// If streaming, close the response body
	if stream && stats.Body != nil {
		stats.Body.Close()
	}

	return stats, nil
}

// GetContainerLogs retrieves logs for a container
func (dc *DockerClient) GetContainerLogs(containerID string, options *LogOptions) (<-chan *LogEntry, <-chan error, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, nil, ErrDockerNotConnected
	}

	if options == nil {
		options = &LogOptions{
			TailStr:  "100",
			Follow:    false,
			Timestamps: false,
		}
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 60*time.Second)
	defer cancel()

	var sinceTime time.Time
	if options.SinceTime != nil {
		sinceTime = *options.SinceTime
	}

	reader, err := client.ContainerLogs(ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Since:      sinceTime.Format(time.RFC3339),
		Tail:       options.TailStr,
		Follow:     options.Follow,
		Timestamps: options.Timestamps,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get logs for container %s: %w", containerID, err)
	}

	logChan := make(chan *LogEntry)
	errChan := make(chan error, 1)

	go parseLogs(reader, logChan, errChan)

	return logChan, errChan, nil
}

// parseLogs parses Docker log output
func parseLogs(reader interface{}, logChan chan<- *LogEntry, errChan chan<- error) {
	defer close(logChan)
	defer close(errChan)

	switch r := reader.(type) {
	case *client.BodyWrapper:
		scanner := &lineScanner{
			reader: r,
			buf:    make([]byte, 4096),
		}
		for {
			line, err := scanner.Scan()
			if err != nil {
				errChan <- err
				return
			}
			if len(line) > 0 {
				logChan <- &LogEntry{
					Timestamp: time.Now(),
					Content:   string(line),
					Stream:    "stdout",
				}
			}
		}
	case types.Reader:
		scanner := &lineScanner{
			reader: r,
			buf:    make([]byte, 4096),
		}
		for {
			line, err := scanner.Scan()
			if err != nil {
				errChan <- err
				return
			}
			if len(line) > 0 {
				logChan <- &LogEntry{
					Timestamp: time.Now(),
					Content:   string(line),
					Stream:    "stdout",
				}
			}
		}
	default:
		errChan <- fmt.Errorf("unsupported reader type: %T", reader)
	}
}

// lineScanner is a simple scanner for reading lines
type lineScanner struct {
	reader interface{}
	buf    []byte
}

// Scan reads the next line
func (s *lineScanner) Scan() ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// StartContainer starts a stopped container
func (dc *DockerClient) StartContainer(containerID string) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if err := client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container %s: %w", containerID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
	}).Info("Container started")

	return nil
}

// StopContainer stops a running container
func (dc *DockerClient) StopContainer(containerID string, timeout time.Duration) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	var timeoutSecs int
	if timeout > 0 {
		timeoutSecs = int(timeout.Seconds())
	}

	if err := client.ContainerStop(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	}); err != nil {
		return fmt.Errorf("failed to stop container %s: %w", containerID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
	}).Info("Container stopped")

	return nil
}

// RestartContainer restarts a container
func (dc *DockerClient) RestartContainer(containerID string, timeout time.Duration) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	var timeoutSecs int
	if timeout > 0 {
		timeoutSecs = int(timeout.Seconds())
	}

	if err := client.ContainerRestart(ctx, containerID, container.StopOptions{
		Timeout: &timeoutSecs,
	}); err != nil {
		return fmt.Errorf("failed to restart container %s: %w", containerID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
	}).Info("Container restarted")

	return nil
}

// RemoveContainer removes a container
func (dc *DockerClient) RemoveContainer(containerID string, force bool) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if err := client.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: force,
	}); err != nil {
		return fmt.Errorf("failed to remove container %s: %w", containerID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
	}).Info("Container removed")

	return nil
}

// CreateContainer creates a new container
func (dc *DockerClient) CreateContainer(config *container.Config, hostConfig *container.HostConfig, networkConfig *network.NetworkingConfig, containerName string) (string, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return "", ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	resp, err := client.ContainerCreate(ctx, config, hostConfig, networkConfig, nil, containerName)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerName,
		"id":        resp.ID,
	}).Info("Container created")

	return resp.ID, nil
}

// ListImages retrieves all images from Docker
func (dc *DockerClient) ListImages(filters map[string][]string) ([]image.Summary, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	opts := image.ListOptions{
		Filters: filterArgsToFilters(filters),
	}

	images, err := client.ImageList(ctx, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list images: %w", err)
	}

	return images, nil
}

// GetImageDetails retrieves detailed information about a specific image
func (dc *DockerClient) GetImageDetails(imageID string) (types.ImageInspect, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return types.ImageInspect{}, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	img, _, err := client.ImageInspectWithRaw(ctx, imageID)
	if err != nil {
		return types.ImageInspect{}, fmt.Errorf("failed to inspect image %s: %w", imageID, err)
	}

	return img, nil
}

// PullImage pulls a new image from a registry
func (dc *DockerClient) PullImage(imageName string) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 5*time.Minute)
	defer cancel()

	opts := image.PullOptions{
		All: false,
	}

	response, err := client.ImagePull(ctx, imageName, opts)
	if err != nil {
		return fmt.Errorf("failed to pull image %s: %w", imageName, err)
	}
	defer response.Close()

	// Wait for pull to complete by reading the response
	_, _ = io.ReadAll(response)

	dc.logger.WithFields(logrus.Fields{
		"image": imageName,
	}).Info("Image pulled")

	return nil
}

// RemoveImage removes an image
func (dc *DockerClient) RemoveImage(imageID string, force bool) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if _, err := client.ImageRemove(ctx, imageID, image.RemoveOptions{
		Force: force,
	}); err != nil {
		return fmt.Errorf("failed to remove image %s: %w", imageID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"image": imageID,
	}).Info("Image removed")

	return nil
}

// ListNetworks retrieves all Docker networks
func (dc *DockerClient) ListNetworks() ([]types.NetworkResource, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	networks, err := client.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list networks: %w", err)
	}

	return networks, nil
}

// GetNetwork retrieves a specific network
func (dc *DockerClient) GetNetwork(networkID string) (types.NetworkResource, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return types.NetworkResource{}, ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	network, err := client.NetworkInspect(ctx, networkID, network.InspectOptions{})
	if err != nil {
		return types.NetworkResource{}, fmt.Errorf("failed to inspect network %s: %w", networkID, err)
	}

	return network, nil
}

// CreateNetwork creates a new Docker network
func (dc *DockerClient) CreateNetwork(config types.NetworkCreate) (string, error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return "", ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	resp, err := client.NetworkCreate(ctx, config.Name, config)
	if err != nil {
		return "", fmt.Errorf("failed to create network: %w", err)
	}

	dc.logger.WithFields(logrus.Fields{
		"network": config.Name,
		"id":      resp.ID,
	}).Info("Network created")

	return resp.ID, nil
}

// RemoveNetwork removes a Docker network
func (dc *DockerClient) RemoveNetwork(networkID string) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if err := client.NetworkRemove(ctx, networkID); err != nil {
		return fmt.Errorf("failed to remove network %s: %w", networkID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"network": networkID,
	}).Info("Network removed")

	return nil
}

// ConnectContainerToNetwork connects a container to a network
func (dc *DockerClient) ConnectContainerToNetwork(containerID, networkID string, config *network.EndpointSettings) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	opts := network.ConnectOptions{
		Container:      containerID,
		EndpointConfig: config,
	}

	if err := client.NetworkConnect(ctx, networkID, opts); err != nil {
		return fmt.Errorf("failed to connect container to network: %w", err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
		"network":   networkID,
	}).Info("Container connected to network")

	return nil
}

// DisconnectContainerFromNetwork disconnects a container from a network
func (dc *DockerClient) DisconnectContainerFromNetwork(containerID, networkID string, force bool) error {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return ErrDockerNotConnected
	}

	ctx, cancel := context.WithTimeout(dc.ctx, 30*time.Second)
	defer cancel()

	if err := client.NetworkDisconnect(ctx, networkID, containerID, force); err != nil {
		return fmt.Errorf("failed to disconnect container from network: %w", containerID, err)
	}

	dc.logger.WithFields(logrus.Fields{
		"container": containerID,
		"network":   networkID,
	}).Info("Container disconnected from network")

	return nil
}

// GetEvents listens to Docker events (streaming)
func (dc *DockerClient) GetEvents(ctx context.Context, since string, until time.Time) (<-chan types.EventsMessage, <-chan error) {
	dc.mu.RLock()
	client := dc.client
	dc.mu.RUnlock()

	if client == nil {
		return nil, nil
	}

	eventCtx, cancel := context.WithCancel(ctx)

	opts := types.EventsOptions{
		Since: since,
		Until: until,
	}

	eventChan := make(chan types.EventsMessage, 100)
	errChan := make(chan error, 1)

	go func() {
		eventsResp, err := client.Events(eventCtx, opts)
		if err != nil {
			errChan <- err
			return
		}
		defer eventsResp.Close()

		decoder := json.NewDecoder(eventsResp.Body)
		for {
			select {
			case <-eventCtx.Done():
				close(eventChan)
				return
			default:
				var event types.EventsMessage
				if err := decoder.Decode(&event); err != nil {
					errChan <- err
					return
				}
				eventChan <- event
			}
		}
	}()

	return eventChan, errChan
}

// StartContainerWatcher starts watching container events for real-time updates
func (dc *DockerClient) StartContainerWatcher(onEvent func(event *Event)) {
	go func() {
		eventCtx, cancel := context.WithCancel(dc.ctx)
		defer cancel()

		since := ""

		for {
			select {
			case <-eventCtx.Done():
				return
			default:
				eventChan, errChan := dc.GetEvents(eventCtx, since, time.Time{})

				if eventChan != nil {
					for {
						select {
						case <-eventCtx.Done():
							return
						case eventMsg, ok := <-eventChan:
							if !ok {
								continue
							}

							since = eventMsg.Actor.Attributes["id"]

							event := dc.convertEvent(&eventMsg)
							if onEvent != nil {
								onEvent(event)
							}
						case err := <-errChan:
							dc.logger.WithError(err).Warn("Error receiving Docker events")
							time.Sleep(5 * time.Second)
							continue
						}
					}
				}

				time.Sleep(1 * time.Second)
			}
		}
	}()
}

// convertEvent converts a Docker events.Message to our Event type
func (dc *DockerClient) convertEvent(event *types.EventsMessage) *Event {
	e := &Event{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	actor := event.Actor

	// Set metadata
	e.Metadata["type"] = event.Type
	e.Metadata["action"] = event.Action
	e.Metadata["actor_id"] = actor.ID
	e.Metadata["actor_type"] = actor.Type

	for k, v := range actor.Attributes {
		e.Metadata[k] = v
	}

	// Map Docker events to our event types
	switch event.Type {
	case "container":
		switch event.Action {
		case "start":
			e.Type = EventContainerStarted
			e.Container = &Container{ID: actor.ID}
		case "stop":
			e.Type = EventContainerStopped
			e.Container = &Container{ID: actor.ID}
		case "die":
			e.Type = EventContainerStopped
			e.Container = &Container{ID: actor.ID}
		case "destroy":
			e.Type = EventContainerStopped
			e.Container = &Container{ID: actor.ID}
		case "rename":
			e.Type = EventContainerStopped // Treat as status change
			e.Container = &Container{ID: actor.ID}
		}
	case "image":
		switch event.Action {
		case "pull":
			e.Type = EventImagePullled
			e.Image = &Image{ID: actor.ID}
		case "untag":
			e.Type = EventImageDeleted
			e.Image = &Image{ID: actor.ID}
		case "delete":
			e.Type = EventImageDeleted
			e.Image = &Image{ID: actor.ID}
		}
	case "network":
		switch event.Action {
		case "create":
			e.Type = EventNetworkCreated
			e.Network = &Network{Name: actor.Attributes["name"]}
		case "disconnect":
			e.Type = EventNetworkDeleted
			e.Network = &Network{Name: actor.Attributes["name"]}
		case "destroy":
			e.Type = EventNetworkDeleted
			e.Network = &Network{Name: actor.Attributes["name"]}
		}
	}

	return e
}

// Shutdown stops the Docker client
func (dc *DockerClient) Shutdown() {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	if dc.cancel != nil {
		dc.cancel()
	}

	if dc.client != nil {
		_ = dc.client.Close()
	}

	dc.initialized = false
	dc.client = nil

	dc.logger.Info("Docker client shut down")
}

// GetClient returns the underlying Docker client
func (dc *DockerClient) GetClient() *client.Client {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.client
}

// filterArgsToFilters converts filter args to types.Filters
func filterArgsToFilters(filters map[string][]string) types.Filters {
	filter := types.NewArgs()
	for key, values := range filters {
		for _, value := range values {
			filter.Add(key, value)
		}
	}
	return filter
}

// GetContainerHealth checks the health status of a container
func (dc *DockerClient) GetContainerHealth(containerID string) (Health, error) {
	inspect, err := dc.GetContainerDetails(containerID)
	if err != nil {
		return Health{}, err
	}

	if inspect.State.Health == nil {
		return Health{
			Status:        "none",
			FailingStreak: 0,
			Log:           []HealthLogEntry{},
		}, nil
	}

	logEntries := make([]HealthLogEntry, len(inspect.State.Health.Log))
	for i, entry := range inspect.State.Health.Log {
		logEntries[i] = HealthLogEntry{
			Start:    entry.Start,
			End:      entry.End,
			Log:      entry.Log,
			ExitCode: entry.ExitCode,
			Output:   strings.Join(entry.Log, "\n"),
		}
	}

	return Health{
		Status:        inspect.State.Health.Status,
		FailingStreak: inspect.State.Health.FailingStreak,
		Log:           logEntries,
	}, nil
}

// EnableHotReload enables hot reload for container status changes
func (dc *DockerClient) EnableHotReload() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.hotReload = true
}

// DisableHotReload disables hot reload
func (dc *DockerClient) DisableHotReload() {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.hotReload = false
}

// IsHotReloadEnabled returns whether hot reload is enabled
func (dc *DockerClient) IsHotReloadEnabled() bool {
	dc.mu.RLock()
	defer dc.mu.RUnlock()
	return dc.hotReload
}
