package catalog

import (
	"encoding/json"
	"time"

	"github.com/sirupsen/logrus"
)

// ServiceStatus represents the health status of a service
type ServiceStatus string

const (
	StatusRunning    ServiceStatus = "running"
	StatusStarting   ServiceStatus = "starting"
	StatusStopped    ServiceStatus = "stopped"
	StatusFailed     ServiceStatus = "failed"
	StatusUnhealthy  ServiceStatus = "unhealthy"
	StatusUnknown    ServiceStatus = "unknown"
	StatusRestarting ServiceStatus = "restarting"
)

// Service represents a discovered service
type Service struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Status       string            `json:"status"`
	HealthStatus string            `json:"health_status"`
	Image        string            `json:"image"`
	Ports        map[string]string `json:"ports"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Namespace    string            `json:"namespace"`
	Created      time.Time         `json:"created"`
	LastUpdated  time.Time         `json:"last_updated"`
	Details      map[string]interface{} `json:"details"`
	Metrics      *ContainerStats   `json:"metrics,omitempty"`
	Health       *Health           `json:"health,omitempty"`
	Age          string            `json:"age"`
	Conditions   []ResourceCondition `json:"conditions"`
}

// Container represents a Docker container in the catalog
type Container struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Image        string            `json:"image"`
	Status       ServiceStatus     `json:"status"`
	State        string            `json:"state"`
	Ports        []Port            `json:"ports"`
	Networks     []string          `json:"networks"`
	Labels       map[string]string `json:"labels"`
	Environment  map[string]string `json:"environment"`
	Created      time.Time         `json:"created"`
	StartedAt    time.Time         `json:"started_at"`
	HealthStatus string            `json:"health_status"`
	Host         string            `json:"host"`
	NodeName     string            `json:"node_name,omitempty"`
	CgroupID     string            `json:"cgroup_id,omitempty"`
	Pids         []string          `json:"pids,omitempty"`
	SysInitPresent bool              `json:"sys_init_present"`
	RootFS       string            `json:"rootfs,omitempty"`
	OpenStdin    bool              `json:"open_stdin"`
	StdinOnce    bool              `json:"stdin_once"`
	Isolation    string            `json:"isolation,omitempty"`
	Health       *Health           `json:"health,omitempty"`
	Stats        *ContainerStats   `json:"stats,omitempty"`
	Resources    ContainerResources `json:"resources,omitempty"`
	Uptime       string            `json:"uptime,omitempty"`
}

// Port represents a container port
type Port struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"host_ip"`
	IP            string `json:"ip"`
	PrivatePort   int    `json:"private_port"`
	PublicPort    int    `json:"public_port"`
	Type          string `json:"type"`
}

// Mount represents a container mount
type Mount struct {
	Type             string `json:"type"`
	Name             string `json:"name"`
	Source           string `json:"source"`
	Destination      string `json:"destination"`
	Driver           string `json:"driver"`
	Mode             string `json:"mode"`
	RW               bool   `json:"rw"`
	Propagation      string `json:"propagation"`
}

// PortMapping represents a port mapping (deprecated, use Port instead)
type PortMapping struct {
	IP         string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort int    `json:"PublicPort"`
	Type       string `json:"Type"`
}

// Network represents a Docker network
type Network struct {
	Name        string            `json:"name"`
	ID          string            `json:"id"`
	Scope       string            `json:"scope"`
	Type        string            `json:"type"`
	Driver      string            `json:"driver"`
	IPAM        NetworkIPAM       `json:"ipam"`
	Labels      map[string]string `json:"labels"`
	Containers  map[string]*NetworkContainer `json:"containers"`
	Created     time.Time         `json:"created"`
	Internal    bool              `json:"internal"`
	EnableIPv6  bool              `json:"enable_ipv6"`
	Attachable  bool              `json:"attachable"`
	Ingress     bool              `json:"ingress"`
	ConfigFrom  map[string]interface{} `json:"config_from"`
	ConfigOnly  bool              `json:"config_only"`
	ContainersCount int           `json:"containers_count"`
}

// Health represents container health status
type Health struct {
	Status        string            `json:"Status"`
	FailingStreak int               `json:"FailingStreak"`
	Log           []HealthLogEntry  `json:"Log"`
}

// HealthLogEntry represents a single health check log entry
type HealthLogEntry struct {
	Start    time.Time `json:"Start"`
	End      time.Time `json:"End"`
	Log      []string  `json:"Log"`
	ExitCode int       `json:"ExitCode"`
	Output   string    `json:"Output"`
}

// ContainerStats represents resource usage statistics
type ContainerStats struct {
	ContainerID     string                 `json:"container_id"`
	Labels          map[string]string      `json:"labels"`
	StorageDriver   string                 `json:"storage_driver"`
	PIDs            int                    `json:"pids"`
	Offline         bool                   `json:"offline"`
	ReadTime        time.Time              `json:"read_time"`
	NumProcesses    int                    `json:"num_processes"`
	BlkioReadBytes  uint64                 `json:"blkio_read_bytes"`
	MemoryUsage     uint64                 `json:"memory_usage"`
	BlkioIOWeight   uint64                 `json:"blkio_io_weight"`
	PercpuUsage     []uint64               `json:"percpu_usage"`
	NumTasks        int                    `json:"num_tasks"`
	ThrottlingData  map[string]interface{} `json:"throttling_data"`
	OnlineCPUs      uint32                 `json:"online_cpus"`
	BlkioWriteBytes uint64                 `json:"blkio_write_bytes"`
	BlkioReadTime   uint64                 `json:"blkio_read_time"`
	CPUUsage        map[string]interface{} `json:"cpu_usage"`
	MemStats        map[string]interface{} `json:"mem_stats"`
}

// ContainerResources represents container resource limits
type ContainerResources struct {
	CPUQuota      int64 `json:"cpu_quota"`
	CPUPeriod     int64 `json:"cpu_period"`
	CPUShares     int64 `json:"cpu_shares"`
	Memory        int64 `json:"memory"`
	MemorySwap    int64 `json:"memory_swap"`
	CPUS        int   `json:"cpus"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	IOMaximumBandwidth int64 `json:"io_max_bandwidth"`
	IOMaximumIOps  int64 `json:"io_max_iops"`
}

// Image represents a Docker image
type Image struct {
	ID              string            `json:"id"`
	RepoTags        []string          `json:"repo_tags"`
	Created         time.Time         `json:"created"`
	Size            int64             `json:"size"`
	VirtualSize     int64             `json:"virtual_size"`
	Labels          map[string]string `json:"labels"`
	Digest          string            `json:"digest"`
	Parent          string            `json:"parent"`
	Architecture    string            `json:"architecture"`
	Os              string            `json:"os"`
	OsVersion       string            `json:"os_version,omitempty"`
	OsFeatures      []string          `json:"os_features,omitempty"`
}

// Network represents a Docker network
type Network struct {
	Name   string            `json:"name"`
	ID     string            `json:"id"`
	Scope  string            `json:"scope"`
	Type   string            `json:"type"`
	Driver string            `json:"driver"`
	IPAM   NetworkIPAM       `json:"ipam"`
	Labels map[string]string `json:"labels"`
	Containers map[string]*NetworkContainer `json:"containers"`
	Created time.Time `json:"created"`
	Internal bool `json:"internal"`
	EnableIPv6 bool `json:"enable_ipv6"`
	Attachable bool `json:"attachable"`
	Ingress bool `json:"ingress"`
	ConfigFrom map[string]interface{} `json:"config_from"`
	ConfigOnly bool `json:"config_only"`
	ContainersCount int `json:"containers_count"`
}

// NetworkIPAM represents network IPAM configuration
type NetworkIPAM struct {
	Driver string            `json:"driver"`
	Config []IPAMConfig      `json:"config"`
	Options map[string]string `json:"options"`
}

// IPAMConfig represents IPAM configuration
type IPAMConfig struct {
	Subnet  string `json:"subnet"`
	IPRange string `json:"ip_range"`
	Gateway string `json:"gateway"`
}

// NetworkContainer represents a container attached to a network
type NetworkContainer struct {
	Name       string            `json:"name"`
	ID         string            `json:"id"`
	IPv4Address string           `json:"ipv4_address"`
	IPv6Address string           `json:"ipv6_address"`
	Links      []string           `json:"links"`
	Aliases    []string           `json:"aliases"`
}

// Kubernetes Resource Types
const (
	ResourceTypePod            = "pod"
	ResourceTypeDeployment     = "deployment"
	ResourceTypeStatefulSet    = "statefulset"
	ResourceTypeDaemonSet      = "daemonset"
	ResourceTypeReplicaSet     = "replicaset"
	ResourceTypeService        = "service"
	ResourceTypeIngress        = "ingress"
	ResourceTypeConfigMap      = "configmap"
	ResourceTypeSecret         = "secret"
	ResourceTypePersistentVolumeClaim = "persistentvolumeclaim"
	ResourceTypePodDisruptionBudget = "poddisruptionbudget"
	ResourceTypeHorizontalPodAutoscaler = "horizontalpodautoscaler"
	ResourceTypeNamespace      = "namespace"
)

// KubernetesResourceType represents a type of K8s resource
type KubernetesResourceType string

// ServiceResource represents a Kubernetes service resource
type ServiceResource struct {
	Type      KubernetesResourceType `json:"type"`
	Name      string                 `json:"name"`
	Namespace string                 `json:"namespace"`
	Labels    map[string]string      `json:"labels"`
	Annotations map[string]string    `json:"annotations"`
	Status    string                 `json:"status"`
	Condition string                 `json:"condition"`
	Age       string                 `json:"age"`
	Details   map[string]interface{} `json:"details"`
	Replicas  int                    `json:"replicas"`
	ReadyReplicas int                  `json:"ready_replicas"`
	Conditions []ResourceCondition  `json:"conditions"`
}

// ResourceCondition represents a resource condition
type ResourceCondition struct {
	Type              string `json:"type"`
	Status            string `json:"status"`
	LastTransitionTime string `json:"last_transition_time"`
	Reason            string `json:"reason"`
	Message           string `json:"message"`
}

// Pod represents a Kubernetes pod
type Pod struct {
	ServiceResource
	Pods         []PodDetail           `json:"pods"`
	Controller   string                `json:"controller"`
	ControllerID string                `json:"controller_id"`
}

// PodDetail represents details of a single pod
type PodDetail struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	Ready        string            `json:"ready"`
	Restarts     int               `json:"restarts"`
	Age          string            `json:"age"`
	PodIP        string            `json:"pod_ip"`
	Node         string            `json:"node"`
	StartedAt    time.Time         `json:"started_at"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Containers   []ContainerDetail `json:"containers"`
	Events       []K8sEvent        `json:"events"`
	OwnerRefs    []OwnerReference  `json:"owner_refs"`
}

// ContainerDetail represents a container in a pod
type ContainerDetail struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	State        string `json:"state"`
	Ready        bool   `json:"ready"`
	Restarts     int    `json:"restarts"`
	LastState    map[string]interface{} `json:"last_state,omitempty"`
	LastTerminatedReason string `json:"last_terminated_reason,omitempty"`
	ReadyReason  string `json:"ready_reason,omitempty"`
}

// OwnerReference represents an owner reference
type OwnerReference struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	UID             string `json:"uid"`
	APIVersion      string `json:"apiVersion"`
	Controller      *bool  `json:"controller,omitempty"`
	BlockOwnerDeletion *bool `json:"block_owner_deletion,omitempty"`
}

// K8sEvent represents a Kubernetes event
type K8sEvent struct {
	Type      string    `json:"type"`
	Reason    string    `json:"reason"`
	Message   string    `json:"message"`
	Source    K8sEventSource `json:"source"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp time.Time `json:"last_timestamp"`
	Count     int       `json:"count"`
	InvolvedObject map[string]string `json:"involved_object"`
	Severity   string    `json:"severity"`
}

// K8sEventSource represents the source of a K8s event
type K8sEventSource struct {
	Component string `json:"component"`
	Host      string `json:"host,omitempty"`
}

// ServiceDefinition represents a Kubernetes service definition
type ServiceDefinition struct {
	ServiceResource
	Ports    []ServicePort `json:"ports"`
	Type     string        `json:"type"`
	ClusterIP string       `json:"cluster_ip"`
	ExternalIPs []string      `json:"external_ips"`
	LoadBalancerIP string    `json:"load_balancer_ip"`
	SessionAffinity string  `json:"session_affinity"`
	Annotations map[string]string `json:"annotations"`
	IPFamilies []string `json:"ip_families"`
}

// ServicePort represents a service port
type ServicePort struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	TargetPort int    `json:"target_port"`
	Protocol   string `json:"protocol"`
	NodePort   int    `json:"node_port"`
	AppProtocol string `json:"app_protocol"`
}

// PodMetrics represents pod resource metrics
type PodMetrics struct {
	Name       string            `json:"name"`
	Namespace  string            `json:"namespace"`
	Timestamp  time.Time         `json:"timestamp"`
	Window     time.Duration     `json:"window"`
	Containers []ContainerMetrics `json:"containers"`
}

// ContainerMetrics represents container resource metrics
type ContainerMetrics struct {
	Name         string            `json:"name"`
	Timestamp    time.Time         `json:"timestamp"`
	Usage        map[string]string `json:"usage"`
	Metrics      map[string]interface{} `json:"metrics"`
}

// Deployment represents a Kubernetes deployment
type Deployment struct {
	ServiceResource
	Strategy   DeploymentStrategy `json:"strategy"`
	Replicas   int                `json:"replicas"`
	CurrentReplicas int          `json:"current_replicas"`
	UpdatedReplicas int         `json:"updated_replicas"`
	AvailableReplicas int     `json:"available_replicas"`
	UnavailableReplicas int `json:"unavailable_replicas"`
	PodTemplate PodTemplateSpec `json:"pod_template"`
	Observation map[string]interface{} `json:"observation"`
}

// DeploymentStrategy represents deployment strategy
type DeploymentStrategy struct {
	Type          string            `json:"type"`
	RollingUpdate *RollingUpdate    `json:"rolling_update,omitempty"`
}

// RollingUpdate represents rolling update parameters
type RollingUpdate struct {
	MaxUnavailable string `json:"max_unavailable"`
	MaxSurge       string `json:"max_surge"`
}

// PodTemplateSpec represents pod template
type PodTemplateSpec struct {
	Metadata map[string]interface{} `json:"metadata"`
	Spec     map[string]interface{} `json:"spec"`
}

// StatefulSet represents a Kubernetes stateful set
type StatefulSet struct {
	ServiceResource
	Replicas       int                   `json:"replicas"`
	CurrentReplicas int                  `json:"current_replicas"`
	ReadyReplicas   int                  `json:"ready_replicas"`
	UpdatedReplicas int                  `json:"updated_replicas"`
	UpdateStrategy StatefulSetUpdateStrategy `json:"update_strategy"`
	PodServiceName  string               `json:"pod_service_name"`
	CurrentPodID    string               `json:"current_pod_id,omitempty"`
	PersistentVolumeClaimRetentionPolicy string `json:"persistent_volume_claim_retention_policy"`
}

// StatefulSetUpdateStrategy represents stateful set update strategy
type StatefulSetUpdateStrategy struct {
	Type           string              `json:"type"`
	RollingUpdate  *RollingUpdateStatefulSet `json:"rolling_update,omitempty"`
	PodManagementPolicy string        `json:"pod_management_policy"`
}

// RollingUpdateStatefulSet represents stateful set rolling update
type RollingUpdateStatefulSet struct {
	Partition     int `json:"partition"`
	MaxUnavailable int `json:"max_unavailable,omitempty"`
}

// DaemonSet represents a Kubernetes daemon set
type DaemonSet struct {
	ServiceResource
	CurrentNumberScheduled int `json:"current_number_scheduled"`
	DesiredNumberScheduled int `json:"desired_number_scheduled"`
	NumberReady            int `json:"number_ready"`
	NumberAvailable        int `json:"number_available"`
	NumberUnavailable      int `json:"number_unavailable"`
	UpdateStrategy         DaemonSetUpdateStrategy `json:"update_strategy"`
}

// DaemonSetUpdateStrategy represents daemon set update strategy
type DaemonSetUpdateStrategy struct {
	Type           string              `json:"type"`
	RollingUpdate  *RollingUpdate      `json:"rolling_update,omitempty"`
}

// ReplicaSet represents a Kubernetes replica set
type ReplicaSet struct {
	ServiceResource
	Replicas          int           `json:"replicas"`
	AvailableReplicas int           `json:"available_replicas"`
	ReadyReplicas     int           `json:"ready_replicas"`
	AvailableStatus   string        `json:"available_status"`
	Conditions        []ResourceCondition `json:"conditions"`
}

// Ingress represents a Kubernetes ingress
type Ingress struct {
	ServiceResource
	Rules    []IngressRule   `json:"rules"`
	LoadBalancer IngressLoadBalancer `json:"load_balancer"`
	TLS      []IngressTLS      `json:"tls"`
}

// IngressRule represents an ingress rule
type IngressRule struct {
	Host string                `json:"host"`
	HTTP *IngressHTTPRule `json:"http,omitempty"`
}

// IngressHTTPRule represents HTTP-specific ingress rules
type IngressHTTPRule struct {
	Paths []IngressPath `json:"paths"`
}

// IngressPath represents an HTTP path rule
type IngressPath struct {
	Path     string `json:"path"`
	PathType string `json:"path_type"`
	Backend  IngressBackend `json:"backend"`
}

// IngressBackend represents an ingress backend
type IngressBackend struct {
	ServiceName string `json:"service_name"`
	ServicePort int    `json:"service_port"`
}

// IngressLoadBalancer represents ingress load balancer status
type IngressLoadBalancer struct {
	Ingress []IngressLoadBalancerEntry `json:"ingress"`
}

// IngressLoadBalancerEntry represents load balancer entry
type IngressLoadBalancerEntry struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
}

// IngressTLS represents ingress TLS configuration
type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secret_name"`
}

// ConfigMap represents a Kubernetes config map
type ConfigMap struct {
	ServiceResource
	Data      map[string]string `json:"data"`
	BinaryData map[string]string `json:"binary_data"`
}

// Secret represents a Kubernetes secret
type Secret struct {
	ServiceResource
	Type       string            `json:"type"`
	Data       map[string]string `json:"data"`
	StringData map[string]string `json:"string_data"`
	Immutable  *bool             `json:"immutable"`
}

// Namespace represents a Kubernetes namespace
type Namespace struct {
	ServiceResource
	Status    string `json:"status"`
	CreationTimestamp time.Time `json:"creation_timestamp"`
	Phase     string `json:"phase"`
}

// PersistentVolumeClaim represents a PVC
type PersistentVolumeClaim struct {
	ServiceResource
	AccessModes []string        `json:"access_modes"`
	Capacity    CapacityInfo    `json:"capacity"`
	Source      PVSource        `json:"source"`
	StorageClassName string `json:"storage_class_name"`
	Status      PvcStatus       `json:"status"`
	VolumeMode  string          `json:"volume_mode"`
}

// CapacityInfo represents capacity information
type CapacityInfo struct {
	Storage string `json:"storage"`
}

// PVSource represents the source of a PVC
type PVSource struct {
	PersistentVolumeName string `json:"persistent_volume_name"`
}

// PvcStatus represents PVC status
type PvcStatus struct {
	Phase string `json:"phase"`
	Conditions []ResourceCondition `json:"conditions"`
}

// PodDisruptionBudget represents PDB
type PodDisruptionBudget struct {
	ServiceResource
	Replicas          int `json:"replicas"`
	DisruptionBudget  string `json:"disruption_budget"`
	CurrentHealthy   int `json:"current_healthy"`
	DesiredHealthy   int `json:"desired_healthy"`
	Status           string `json:"status"`
	Events           []K8sEvent `json:"events"`
}

// HorizontalPodAutoscaler represents HPA
type HorizontalPodAutoscaler struct {
	ServiceResource
	MinReplicas   *int  `json:"min_replicas"`
	MaxReplicas   int   `json:"max_replicas"`
	CurrentReplicas int `json:"current_replicas"`
	DesiredReplicas int `json:"desired_replicas"`
	CPUUtilization *HPAUtilization `json:"cpu_utilization"`
	Conditions []ResourceCondition `json:"conditions"`
	LastScaleTime *time.Time `json:"last_scale_time"`
}

// HPAUtilization represents CPU utilization metrics for HPA
type HPAUtilization struct {
	Percentage int `json:"percentage"`
	AverageUtilization int `json:"average_utilization"`
}

// MetricsCollector defines the interface for collecting service metrics
type MetricsCollector interface {
	CollectionName() string
	CollectContainers() ([]*Container, error)
	CollectImages() ([]*Image, error)
	CollectNetworks() ([]*Network, error)
	CollectKubernetesResources() ([]*ServiceResource, error)
}

// ServiceDiscovery defines the interface for service discovery
type ServiceDiscovery interface {
	CollectMetrics() MetricsCollector
	Discovery() (DiscoveryResult, error)
	Discover() (DiscoveryResult, error)
	GetServiceStatus(serviceID string) (ServiceStatus, error)
	HealthCheck(serviceID string) error
	UpdateStatus(serviceID string, status ServiceStatus) error
}

// DiscoveryResult represents the result of a discovery operation
type DiscoveryResult struct {
	Timestamp    time.Time    `json:"timestamp"`
	Containers   []*Container `json:"containers"`
	Images       []*Image     `json:"images"`
	Networks     []*Network   `json:"networks"`
	K8sResources []*ServiceResource `json:"k8s_resources"`
	Errors       []string     `json:"errors"`
	Success      bool         `json:"success"`
}

// DiscoveryConfig contains configuration for service discovery
type DiscoveryConfig struct {
	EnableDocker     bool          `json:"enable_docker"`
	EnableKubernetes bool          `json:"enable_kubernetes"`
	RefreshInterval  time.Duration `json:"refresh_interval"`
	HealthCheckTimeout time.Duration `json:"health_check_timeout"`
	DockerSocket     string        `json:"docker_socket"`
	KubeconfigPath   string        `json:"kubeconfig_path"`
	Namespaces       []string      `json:"namespaces"`
	ResourceTypes    []string      `json:"resource_types"`
	IncludeMetrics   bool          `json:"include_metrics"`
	EnableEvents     bool          `json:"enable_events"`
}

// DiscoveryMetrics represents discovery metrics
type DiscoveryMetrics struct {
	Timestamp         time.Time `json:"timestamp"`
	ContainersFound   int       `json:"containers_found"`
	ImagesFound       int       `json:"images_found"`
	NetworksFound     int       `json:"networks_found"`
	K8sResourcesFound int       `json:"k8s_resources_found"`
	Duration          int64     `json:"duration_ms"`
	Errors            int       `json:"errors"`
	LastSuccessfulRun time.Time `json:"last_successful_run"`
	LastFailedRun     time.Time `json:"last_failed_run"`
}

// Event defines the structure for catalog events
type Event struct {
	Type        EventType              `json:"type"`
	ID          string                 `json:"id,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Service     *Service               `json:"service,omitempty"`
	Container   *Container             `json:"container,omitempty"`
	Image       *Image                 `json:"image,omitempty"`
	Network     *Network               `json:"network,omitempty"`
	Pod         *PodDetail             `json:"pod,omitempty"`
	Deployment  *ServiceResource       `json:"deployment,omitempty"`
	ServiceObj  *ServiceResource       `json:"service,omitempty"`
	K8sResource *ServiceResource       `json:"k8s_resource,omitempty"`
	Error       error                  `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// EventType represents the type of catalog event
type EventType string

const (
	// Service events
	EventServiceAdded EventType = "service.added"
	EventServiceUpdated EventType = "service.updated"
	EventServiceDeleted EventType = "service.deleted"

	// Container events
	EventContainerStarted EventType = "container.started"
	EventContainerStopped EventType = "container.stopped"
	EventContainerRestarted EventType = "container.restarted"
	EventContainerHealthChanged EventType = "container.health_changed"
	EventContainerStatusChanged EventType = "container.status_changed"
	EventContainerLogs EventType = "container.logs"

	// Image events
	EventImagePulled EventType = "image.pulled"
	EventImageDeleted EventType = "image.deleted"

	// Network events
	EventNetworkCreated EventType = "network.created"
	EventNetworkDeleted EventType = "network.deleted"

	// K8s resource events
	EventPodAdded EventType = "pod.added"
	EventPodUpdated EventType = "pod.updated"
	EventPodDeleted EventType = "pod.deleted"
	EventDeploymentAdded EventType = "deployment.added"
	EventDeploymentUpdated EventType = "deployment.updated"
	EventDeploymentDeleted EventType = "deployment.deleted"
	EventServiceAdded EventType = "service.added"
	EventServiceUpdated EventType = "service.updated"
	EventServiceDeleted EventType = "service.deleted"

	// Discovery events
	EventDiscoveryCompleted EventType = "discovery.completed"
	EventDiscoveryFailed EventType = "discovery.failed"

	// Health check events
	EventHealthCheckPassed EventType = "health_check.passed"
	EventHealthCheckFailed EventType = "health_check.failed"

	// Error event
	EventError EventType = "error"

	// System events
	EventHeartbeat EventType = "heartbeat"
)

// Subscriber represents an event subscriber
type Subscriber interface {
	HandleEvent(event *Event)
}

// EventBus represents an event bus for service discovery events
type EventBus interface {
	Publish(event *Event)
	Subscribe(eventType EventType, subscriber Subscriber)
	Unsubscribe(eventType EventType, subscriber Subscriber)
	SubscribeAll(subscriber Subscriber)
	UnsubscribeAll(subscriber Subscriber)
}

// Subscription represents an event subscription
type Subscription struct {
	ID          string
	EventType   EventType
	Subscriber  Subscriber
}

// ContainerHealthWatcher watches container health status
type ContainerHealthWatcher interface {
	Watch(containerID string, done <-chan struct{}) error
	Unwatch(containerID string) error
}

// ResourceWatcher watches Kubernetes resource changes
type ResourceWatcher interface {
	Watch(resourceType string, namespace string, done <-chan struct{}) error
	Unwatch(resourceType string, namespace string) error
}

// ResourceStatusTracker tracks status changes for resources
type ResourceStatusTracker interface {
	TrackStatus(resourceType string, name string, status ServiceStatus) error
	GetStatus(resourceType string, name string) (ServiceStatus, error)
	Untrack(resourceType string, name string) error
}

// MetadataExtractor extracts metadata from container labels and environment
type MetadataExtractor interface {
	Extract(*Container) map[string]interface{}
	ExtractK8s(*ServiceResource) map[string]interface{}
}

// LogEntry represents a log entry from a container
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Content   string    `json:"content"`
	Stream    string    `json:"stream"`
	Line      int       `json:"line"`
}

// LogReader provides access to container logs
type LogReader interface {
	Read(containerID string, options *LogOptions) (<-chan *LogEntry, <-chan error, error)
}

// LogOptions represents options for reading container logs
type LogOptions struct {
	Follow    bool
	Since     time.Time
	TailLines *int64
	SinceTime *time.Time
	Timestamps bool
	SinceStr string
	TailStr string
	SinceTimeStr string
}

// ResourceFilter filters resources by criteria
type ResourceFilter interface {
	FilterContainers(containers []*Container) []*Container
	FilterImages(images []*Image) []*Image
	FilterNetworks(networks []*Network) []*Network
	FilterK8sResources(resources []*ServiceResource) []*ServiceResource
}

// FilterOptions represents filter options
type FilterOptions struct {
	Labels map[string]string
	Status []ServiceStatus
	Tags []string
	Namespace string
	ResourceTypes []string
}

// MarshalJSON custom marshaling for ContainerStats
func (c *ContainerStats) MarshalJSON() ([]byte, error) {
	type Alias ContainerStats
	return json.Marshal(&struct {
		*Alias
		ReadTimeStr string `json:"read_time"`
	}{
		Alias: (*Alias)(c),
		ReadTimeStr: c.ReadTime.Format(time.RFC3339),
	})
}

// MarshalJSON custom marshaling for DiscoveryMetrics
func (m *DiscoveryMetrics) MarshalJSON() ([]byte, error) {
	type Alias DiscoveryMetrics
	return json.Marshal(&struct {
		*Alias
		TimestampStr string `json:"timestamp"`
		LastSuccessfulRunStr string `json:"last_successful_run"`
		LastFailedRunStr string `json:"last_failed_run"`
	}{
		Alias: (*Alias)(m),
		TimestampStr: m.Timestamp.Format(time.RFC3339),
		LastSuccessfulRunStr: m.LastSuccessfulRun.Format(time.RFC3339),
		LastFailedRunStr: m.LastFailedRun.Format(time.RFC3339),
	})
}

// UnmarshalJSON custom unmarshaling for DiscoveryMetrics
func (m *DiscoveryMetrics) UnmarshalJSON(data []byte) error {
	type Alias DiscoveryMetrics
	aux := &struct {
		*Alias
		TimestampStr string `json:"timestamp"`
		LastSuccessfulRunStr string `json:"last_successful_run"`
		LastFailedRunStr string `json:"last_failed_run"`
	}{}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	*m = DiscoveryMetrics(*aux.Alias)

	if aux.TimestampStr != "" {
		t, err := time.Parse(time.RFC3339, aux.TimestampStr)
		if err == nil {
			m.Timestamp = t
		}
	}

	if aux.LastSuccessfulRunStr != "" {
		t, err := time.Parse(time.RFC3339, aux.LastSuccessfulRunStr)
		if err == nil {
			m.LastSuccessfulRun = t
		}
	}

	if aux.LastFailedRunStr != "" {
		t, err := time.Parse(time.RFC3339, aux.LastFailedRunStr)
		if err == nil {
			m.LastFailedRun = t
		}
	}

	return nil
}

// String returns a string representation of ServiceStatus
func (s ServiceStatus) String() string {
	return string(s)
}

// ParseServiceStatus parses a service status from string
func ParseServiceStatus(s string) ServiceStatus {
	switch s {
	case "running", "Running", "RUNNING":
		return StatusRunning
	case "starting", "Starting", "STARTING":
		return StatusStarting
	case "stopped", "Stopped", "STOPPED":
		return StatusStopped
	case "failed", "Failed", "FAILED":
		return StatusFailed
	case "unhealthy", "Unhealthy", "UNHEALTHY":
		return StatusUnhealthy
	case "restarting", "Restarting", "RESTARTING":
		return StatusRestarting
	default:
		return StatusUnknown
	}
}

// LogFormat formats container logs for display
type LogFormat struct {
	Prefix string
	Suffix string
	Format string
}

// DefaultLogFormat returns the default log format
func DefaultLogFormat() LogFormat {
	return LogFormat{
		Prefix: "  ",
		Suffix: "\n",
		Format: "[{:.6f}] {stream}: {line}\n",
	}
}

// MetricsFormatter formats metrics for output
type MetricsFormatter interface {
	Format(m *ContainerStats) string
	FormatResource(r *ServiceResource) string
	FormatDiscovery(d *DiscoveryMetrics) string
}

// LogOutput represents log output destination
type LogOutput struct {
	Level logrus.Level
	Format string
}

// DefaultLogOutput returns default log output configuration
func DefaultLogOutput() *LogOutput {
	return &LogOutput{
		Level: logrus.InfoLevel,
		Format: "json",
	}
}

// ResourceCache caches resource information for efficient retrieval
type ResourceCache interface {
	Get(containerID string) (*Container, bool)
	Set(containerID string, container *Container)
	Remove(containerID string)
	AllContainers() []*Container
	GetImage(digest string) (*Image, bool)
	SetImage(digest string, image *Image)
	RemoveImage(digest string)
	AllImages() []*Image
	GetNetwork(id string) (*Network, bool)
	SetNetwork(id string, network *Network)
	RemoveNetwork(id string)
	AllNetworks() []*Network
	GetK8sResource(resourceType, namespace, name string) (*ServiceResource, bool)
	SetK8sResource(resourceType, namespace, name string, resource *ServiceResource)
	RemoveK8sResource(resourceType, namespace, name string)
	AllK8sResources() []*ServiceResource
}

// CacheMetrics represents cache metrics
type CacheMetrics struct {
	ContainerHitRate float64 `json:"container_hit_rate"`
	ImageHitRate float64 `json:"image_hit_rate"`
	NetworkHitRate float64 `json:"network_hit_rate"`
	K8sHitRate float64 `json:"k8s_hit_rate"`
	TotalContainers int `json:"total_containers"`
	TotalImages int `json:"total_images"`
	TotalNetworks int `json:"total_networks"`
	TotalK8sResources int `json:"total_k8s_resources"`
}

// DiscoveryEventHandler handles discovery events
type DiscoveryEventHandler interface {
	OnDiscoveryCompleted(metrics *DiscoveryMetrics)
	OnDiscoveryFailed(err error, metrics *DiscoveryMetrics)
	OnContainerStatusChange(containerID string, status ServiceStatus)
	OnResourceStatusChange(resourceType, name string, status ServiceStatus)
}
