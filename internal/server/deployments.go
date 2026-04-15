package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

type deploymentManager interface {
	Apply(ctx context.Context, req deploymentApplyRequest) (*deploymentRecord, error)
	Status(ctx context.Context, namespace, name string) (*deploymentRecord, error)
}

type commandRunner interface {
	Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error)
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

type kubectlDeploymentManager struct {
	kubectlPath    string
	defaultNS      string
	applyTimeout   time.Duration
	rolloutTimeout time.Duration
	runner         commandRunner
	logger         *logrus.Logger
}

type deploymentApplyRequest struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Image       string `json:"image"`
	Port        int    `json:"port"`
	Replicas    int    `json:"replicas"`
	ServiceType string `json:"service_type"`
	Delivery    string `json:"delivery"`
}

type deploymentRecord struct {
	Name              string         `json:"name"`
	Namespace         string         `json:"namespace"`
	Image             string         `json:"image"`
	Replicas          int            `json:"replicas"`
	ReadyReplicas     int            `json:"ready_replicas"`
	AvailableReplicas int            `json:"available_replicas"`
	UpdatedReplicas   int            `json:"updated_replicas"`
	ServiceType       string         `json:"service_type"`
	ClusterIP         string         `json:"cluster_ip,omitempty"`
	Phase             string         `json:"phase"`
	Conditions        []string       `json:"conditions"`
	Message           string         `json:"message"`
	ManifestName      string         `json:"manifest_name,omitempty"`
	Delivery          string         `json:"delivery,omitempty"`
	RepoURL           string         `json:"repo_url,omitempty"`
	Revision          string         `json:"revision,omitempty"`
	Path              string         `json:"path,omitempty"`
	ApplicationName   string         `json:"application_name,omitempty"`
	SyncStatus        string         `json:"sync_status,omitempty"`
	HealthStatus      string         `json:"health_status,omitempty"`
	ExecutionState    string         `json:"execution_state,omitempty"`
	ExecutionPlan     *executionPlan `json:"execution_plan,omitempty"`
}

func newDeploymentManager(cfg *config.Config, logger *logrus.Logger) deploymentManager {
	if cfg == nil {
		cfg = config.NewConfig()
	}

	kubectlPath := strings.TrimSpace(cfg.KubectlPath)
	if kubectlPath == "" {
		kubectlPath = "kubectl"
	}

	defaultNS := normalizeKubernetesName(cfg.KubernetesNamespace)
	if defaultNS == "" {
		defaultNS = "axiom-apps"
	}

	applyTimeout := cfg.KubernetesApplyTimeout
	if applyTimeout <= 0 {
		applyTimeout = 90 * time.Second
	}

	rolloutTimeout := cfg.KubernetesRolloutTimeout
	if rolloutTimeout <= 0 {
		rolloutTimeout = 180 * time.Second
	}

	return &kubectlDeploymentManager{
		kubectlPath:    kubectlPath,
		defaultNS:      defaultNS,
		applyTimeout:   applyTimeout,
		rolloutTimeout: rolloutTimeout,
		runner:         execCommandRunner{},
		logger:         logger,
	}
}

func (m *kubectlDeploymentManager) Apply(ctx context.Context, req deploymentApplyRequest) (*deploymentRecord, error) {
	spec, err := normalizeDeploymentRequest(req, m.defaultNS)
	if err != nil {
		return nil, err
	}

	if err := m.ensureNamespace(ctx, spec.Namespace); err != nil {
		return nil, err
	}

	manifest := renderDeploymentManifest(spec)
	applyCtx, cancelApply := context.WithTimeout(ctx, m.applyTimeout)
	defer cancelApply()
	if _, err := m.runner.Run(applyCtx, m.kubectlPath, []string{"apply", "-f", "-"}, []byte(manifest)); err != nil {
		return nil, fmt.Errorf("failed to apply deployment manifest: %w", err)
	}

	rolloutCtx, cancelRollout := context.WithTimeout(ctx, m.rolloutTimeout)
	defer cancelRollout()
	if _, err := m.runner.Run(rolloutCtx, m.kubectlPath, []string{"-n", spec.Namespace, "rollout", "status", "deployment/" + spec.Name, "--timeout=" + m.rolloutTimeout.String()}, nil); err != nil {
		return nil, fmt.Errorf("deployment rollout did not complete: %w", err)
	}

	record, err := m.Status(ctx, spec.Namespace, spec.Name)
	if err != nil {
		return nil, err
	}
	record.ManifestName = spec.Name
	record.ExecutionPlan = newDeploymentExecutionPlan("deployment_apply", spec)
	record.ExecutionState = record.Phase
	return record, nil
}

func (m *kubectlDeploymentManager) Status(ctx context.Context, namespace, name string) (*deploymentRecord, error) {
	namespace = normalizeKubernetesName(namespace)
	name = normalizeKubernetesName(name)
	if namespace == "" || name == "" {
		return nil, errors.New("deployment namespace and name are required")
	}

	output, err := m.runner.Run(ctx, m.kubectlPath, []string{"-n", namespace, "get", "deployment", name, "-o", "json"}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch deployment status: %w", err)
	}

	var deployment struct {
		Metadata struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
		} `json:"metadata"`
		Spec struct {
			Replicas int32 `json:"replicas"`
			Template struct {
				Spec struct {
					Containers []struct {
						Image string `json:"image"`
						Ports []struct {
							ContainerPort int32 `json:"containerPort"`
						} `json:"ports"`
					} `json:"containers"`
				} `json:"spec"`
			} `json:"template"`
		} `json:"spec"`
		Status struct {
			ReadyReplicas     int32 `json:"readyReplicas"`
			AvailableReplicas int32 `json:"availableReplicas"`
			UpdatedReplicas   int32 `json:"updatedReplicas"`
			Conditions        []struct {
				Type    string `json:"type"`
				Status  string `json:"status"`
				Reason  string `json:"reason"`
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &deployment); err != nil {
		return nil, fmt.Errorf("failed to parse deployment status: %w", err)
	}

	record := &deploymentRecord{
		Name:              deployment.Metadata.Name,
		Namespace:         deployment.Metadata.Namespace,
		Replicas:          int(deployment.Spec.Replicas),
		ReadyReplicas:     int(deployment.Status.ReadyReplicas),
		AvailableReplicas: int(deployment.Status.AvailableReplicas),
		UpdatedReplicas:   int(deployment.Status.UpdatedReplicas),
		Phase:             deploymentPhase(int(deployment.Spec.Replicas), int(deployment.Status.ReadyReplicas), int(deployment.Status.AvailableReplicas)),
	}

	if len(deployment.Spec.Template.Spec.Containers) > 0 {
		record.Image = deployment.Spec.Template.Spec.Containers[0].Image
	}

	for _, condition := range deployment.Status.Conditions {
		if strings.EqualFold(condition.Status, "true") {
			record.Conditions = append(record.Conditions, condition.Type)
		}
		if record.Message == "" && strings.TrimSpace(condition.Message) != "" {
			record.Message = condition.Message
		}
	}
	if record.Message == "" {
		record.Message = fmt.Sprintf("%s has %d/%d ready replicas", record.Name, record.ReadyReplicas, record.Replicas)
	}
	record.ExecutionState = record.Phase
	record.ExecutionPlan = newDeploymentExecutionPlan("deployment_status", deploymentApplyRequest{
		Namespace: namespace,
		Name:      name,
		Delivery:  "kubectl",
	})

	serviceOutput, err := m.runner.Run(ctx, m.kubectlPath, []string{"-n", namespace, "get", "service", name, "-o", "json"}, nil)
	if err == nil {
		var service struct {
			Spec struct {
				Type      string `json:"type"`
				ClusterIP string `json:"clusterIP"`
			} `json:"spec"`
		}
		if json.Unmarshal(serviceOutput, &service) == nil {
			record.ServiceType = service.Spec.Type
			record.ClusterIP = service.Spec.ClusterIP
		}
	}

	return record, nil
}

func (m *kubectlDeploymentManager) ensureNamespace(ctx context.Context, namespace string) error {
	_, err := m.runner.Run(ctx, m.kubectlPath, []string{"get", "namespace", namespace, "-o", "name"}, nil)
	if err == nil {
		return nil
	}

	if m.logger != nil {
		m.logger.WithField("namespace", namespace).Info("creating deployment namespace")
	}

	_, createErr := m.runner.Run(ctx, m.kubectlPath, []string{"create", "namespace", namespace}, nil)
	if createErr != nil && !strings.Contains(createErr.Error(), "already exists") {
		return fmt.Errorf("failed to create namespace %s: %w", namespace, createErr)
	}
	return nil
}

func normalizeDeploymentRequest(req deploymentApplyRequest, defaultNamespace string) (deploymentApplyRequest, error) {
	req.Name = normalizeKubernetesName(req.Name)
	req.Namespace = normalizeKubernetesName(req.Namespace)
	req.Image = strings.TrimSpace(req.Image)
	req.ServiceType = strings.TrimSpace(req.ServiceType)
	req.Delivery = strings.ToLower(strings.TrimSpace(req.Delivery))

	if req.Name == "" {
		return deploymentApplyRequest{}, errors.New("deployment name is required")
	}
	if req.Namespace == "" {
		req.Namespace = normalizeKubernetesName(defaultNamespace)
	}
	if req.Namespace == "" {
		return deploymentApplyRequest{}, errors.New("deployment namespace is required")
	}
	if req.Image == "" {
		req.Image = "nginx:1.27-alpine"
	}
	if req.Port <= 0 {
		req.Port = defaultPortForImage(req.Image)
	}
	if req.Port < 1 || req.Port > 65535 {
		return deploymentApplyRequest{}, errors.New("deployment port must be between 1 and 65535")
	}
	if req.Replicas <= 0 {
		req.Replicas = 1
	}
	if req.Replicas > 5 {
		return deploymentApplyRequest{}, errors.New("deployment replicas must be between 1 and 5")
	}
	if req.ServiceType == "" {
		req.ServiceType = "ClusterIP"
	}
	if req.ServiceType != "ClusterIP" && req.ServiceType != "NodePort" {
		return deploymentApplyRequest{}, errors.New("deployment service_type must be ClusterIP or NodePort")
	}
	switch req.Delivery {
	case "", "kubectl", "kubernetes":
		req.Delivery = "kubectl"
	case "argocd", "argo", "argo-cd", "github", "github-argocd":
		req.Delivery = "argocd"
	default:
		return deploymentApplyRequest{}, errors.New("deployment delivery must be kubectl or argocd")
	}
	return req, nil
}

func renderDeploymentManifest(req deploymentApplyRequest) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: axiom-idp
spec:
  replicas: %d
  selector:
    matchLabels:
      app.kubernetes.io/name: %s
  template:
    metadata:
      labels:
        app.kubernetes.io/name: %s
        app.kubernetes.io/managed-by: axiom-idp
    spec:
      automountServiceAccountToken: false
      containers:
        - name: %s
          image: %s
          imagePullPolicy: IfNotPresent
          ports:
            - containerPort: %d
          securityContext:
            allowPrivilegeEscalation: false
            seccompProfile:
              type: RuntimeDefault
---
apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
    app.kubernetes.io/managed-by: axiom-idp
spec:
  type: %s
  selector:
    app.kubernetes.io/name: %s
  ports:
    - name: http
      port: %d
      targetPort: %d
`, req.Name, req.Namespace, req.Name, req.Replicas, req.Name, req.Name, req.Name, req.Image, req.Port, req.Name, req.Namespace, req.Name, req.ServiceType, req.Name, req.Port, req.Port)
}

func defaultPortForImage(image string) int {
	lower := strings.ToLower(image)
	switch {
	case strings.Contains(lower, "nginx"), strings.Contains(lower, "httpd"), strings.Contains(lower, "caddy"):
		return 80
	default:
		return 8080
	}
}

var (
	imagePattern      = regexp.MustCompile(`(?:image|using)\s+([a-zA-Z0-9./:_-]+)`)
	namespacePattern  = regexp.MustCompile(`(?:namespace|ns)\s+([a-z0-9-]+)`)
	replicasPattern   = regexp.MustCompile(`(\d+)\s+replicas?`)
	portPattern       = regexp.MustCompile(`port\s+(\d{2,5})`)
	nameHintPattern   = regexp.MustCompile(`(?:named|called|app|application|service|deployment)\s+([a-z0-9][a-z0-9-]{1,41})`)
	statusNamePattern = regexp.MustCompile(`(?:status(?:\s+of)?|deployment(?:\s+status)?(?:\s+for)?|rollout(?:\s+status)?(?:\s+for)?)\s+([a-z0-9][a-z0-9-]{1,41})`)
)

func parseDeploymentPrompt(query string, defaultNamespace string) (deploymentApplyRequest, bool) {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" {
		return deploymentApplyRequest{}, false
	}

	if !strings.HasPrefix(lower, "deploy ") &&
		!containsAny(lower, "deploy application", "deploy app", "create deployment", "rollout application", "launch application", "launch app") {
		return deploymentApplyRequest{}, false
	}

	req := deploymentApplyRequest{
		Namespace:   defaultNamespace,
		ServiceType: "ClusterIP",
	}

	if match := imagePattern.FindStringSubmatch(query); len(match) == 2 {
		req.Image = strings.TrimSpace(match[1])
	}
	if match := namespacePattern.FindStringSubmatch(lower); len(match) == 2 {
		req.Namespace = match[1]
	}
	if match := replicasPattern.FindStringSubmatch(lower); len(match) == 2 {
		if parsed, err := strconv.Atoi(match[1]); err == nil {
			req.Replicas = parsed
		}
	}
	if match := portPattern.FindStringSubmatch(lower); len(match) == 2 {
		if parsed, err := strconv.Atoi(match[1]); err == nil {
			req.Port = parsed
		}
	}
	if strings.Contains(lower, "nodeport") {
		req.ServiceType = "NodePort"
	}
	if match := nameHintPattern.FindStringSubmatch(lower); len(match) == 2 {
		req.Name = match[1]
	}
	if req.Name == "" && req.Image != "" {
		segments := strings.Split(strings.TrimSuffix(req.Image, ":"), "/")
		req.Name = normalizeKubernetesName(strings.Split(segments[len(segments)-1], ":")[0])
	}
	if req.Name == "" {
		req.Name = "sample-web"
	}
	return req, true
}

func parseDeploymentStatusPrompt(query string, defaultNamespace string) (string, string, bool) {
	lower := strings.ToLower(strings.TrimSpace(query))
	if !containsAny(lower, "deployment status", "status of deployment", "rollout status", "status for") {
		return "", "", false
	}
	namespace := defaultNamespace
	if match := namespacePattern.FindStringSubmatch(lower); len(match) == 2 {
		namespace = match[1]
	}
	if match := statusNamePattern.FindStringSubmatch(lower); len(match) == 2 {
		return namespace, match[1], true
	}
	if match := nameHintPattern.FindStringSubmatch(lower); len(match) == 2 {
		return namespace, match[1], true
	}
	return "", "", false
}

func normalizeKubernetesName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "_", "-")
	value = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(value, "-")
	value = strings.Trim(value, "-")
	if len(value) > 42 {
		value = strings.Trim(value[:42], "-")
	}
	return value
}

func deploymentPhase(desired, ready, available int) string {
	switch {
	case desired == 0:
		return "unknown"
	case ready == desired && available == desired:
		return "ready"
	case ready > 0 || available > 0:
		return "progressing"
	default:
		return "pending"
	}
}
