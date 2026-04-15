package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
	"github.com/sirupsen/logrus"
)

type infrastructureApplyRequest struct {
	Name            string            `json:"name"`
	Provider        string            `json:"provider"`
	TargetNamespace string            `json:"target_namespace"`
	Inputs          map[string]string `json:"inputs"`
}

type infrastructureRecord struct {
	Name            string            `json:"name"`
	Provider        string            `json:"provider"`
	TargetNamespace string            `json:"target_namespace"`
	Phase           string            `json:"phase"`
	Message         string            `json:"message"`
	Executed        bool              `json:"executed"`
	ExecutionState  string            `json:"execution_state,omitempty"`
	RepoURL         string            `json:"repo_url,omitempty"`
	Revision        string            `json:"revision,omitempty"`
	Path            string            `json:"path,omitempty"`
	ApplicationName string            `json:"application_name,omitempty"`
	SyncStatus      string            `json:"sync_status,omitempty"`
	HealthStatus    string            `json:"health_status,omitempty"`
	Artifacts       []string          `json:"artifacts,omitempty"`
	Outputs         map[string]string `json:"outputs,omitempty"`
	Inputs          map[string]string `json:"inputs,omitempty"`
	ExecutionPlan   *executionPlan    `json:"execution_plan,omitempty"`
}

type gitOpsOrchestrator struct {
	cfg    *config.Config
	logger *logrus.Logger
}

type gitOpsManager interface {
	ApplyArgoCDDeployment(ctx context.Context, req deploymentApplyRequest) (*deploymentRecord, error)
	ArgoCDDeploymentStatus(ctx context.Context, namespace, name string) (*deploymentRecord, error)
	ApplyInfrastructure(ctx context.Context, req infrastructureApplyRequest) (*infrastructureRecord, error)
	TerraformInfrastructureStatus(ctx context.Context, name string) (*infrastructureRecord, error)
}

type gitOpsContext struct {
	RepoURL         string
	RepoHTTPSURL    string
	BaseBranch      string
	Branch          string
	Path            string
	LocalDir        string
	ApplicationName string
}

func newGitOpsOrchestrator(cfg *config.Config, logger *logrus.Logger) *gitOpsOrchestrator {
	if cfg == nil {
		cfg = config.NewConfig()
	}
	return &gitOpsOrchestrator{cfg: cfg, logger: logger}
}

func (o *gitOpsOrchestrator) ApplyArgoCDDeployment(ctx context.Context, req deploymentApplyRequest) (*deploymentRecord, error) {
	spec, err := normalizeDeploymentRequest(req, o.cfg.KubernetesNamespace)
	if err != nil {
		return nil, err
	}

	gitCtx, err := o.prepareGitOpsBranch(ctx, "deploy", spec.Name)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(gitCtx.LocalDir)

	manifestPath := filepath.Join(gitCtx.LocalDir, filepath.FromSlash(gitCtx.Path))
	if err := os.MkdirAll(manifestPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create manifest path: %w", err)
	}

	files := map[string]string{
		"namespace.yaml":     fmt.Sprintf("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n", spec.Namespace),
		"deployment.yaml":    renderDeploymentOnlyManifest(spec),
		"service.yaml":       renderServiceOnlyManifest(spec),
		"kustomization.yaml": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- namespace.yaml\n- deployment.yaml\n- service.yaml\n",
	}
	if err := writeFiles(manifestPath, files); err != nil {
		return nil, err
	}

	if err := o.commitAndPush(ctx, gitCtx, fmt.Sprintf("Axiom AI deploy %s via Argo CD", spec.Name)); err != nil {
		return nil, err
	}

	if err := o.applyArgoCDApplication(ctx, gitCtx, spec.Namespace); err != nil {
		return nil, err
	}

	if err := o.waitForApplication(ctx, gitCtx.ApplicationName); err != nil {
		return nil, err
	}

	status, err := o.ArgoCDDeploymentStatus(ctx, spec.Namespace, spec.Name)
	if err != nil {
		return nil, err
	}
	status.RepoURL = gitCtx.RepoHTTPSURL
	status.Revision = gitCtx.Branch
	status.Path = gitCtx.Path
	status.ApplicationName = gitCtx.ApplicationName
	status.Delivery = "github-argocd"
	status.ExecutionPlan = newDeploymentExecutionPlan("deployment_apply_argocd", spec)
	status.ExecutionState = status.Phase
	return status, nil
}

func (o *gitOpsOrchestrator) ArgoCDDeploymentStatus(ctx context.Context, namespace, name string) (*deploymentRecord, error) {
	record, err := o.getKubernetesDeploymentStatus(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	appStatus, err := o.getApplicationStatus(ctx, name)
	if err == nil {
		record.SyncStatus = appStatus.SyncStatus
		record.HealthStatus = appStatus.HealthStatus
		record.ApplicationName = appStatus.Name
	}
	record.Delivery = "github-argocd"
	record.ExecutionPlan = newDeploymentExecutionPlan("deployment_status_argocd", deploymentApplyRequest{
		Namespace: namespace,
		Name:      name,
		Delivery:  "argocd",
	})
	record.ExecutionState = record.Phase
	return record, nil
}

func (o *gitOpsOrchestrator) ApplyInfrastructure(ctx context.Context, req infrastructureApplyRequest) (*infrastructureRecord, error) {
	spec, err := normalizeInfrastructureRequest(req)
	if err != nil {
		return nil, err
	}

	gitCtx, err := o.prepareGitOpsBranch(ctx, spec.Provider, spec.Name)
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(gitCtx.LocalDir)

	manifestPath := filepath.Join(gitCtx.LocalDir, filepath.FromSlash(gitCtx.Path))
	if err := os.MkdirAll(manifestPath, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create infra manifest path: %w", err)
	}

	record := &infrastructureRecord{
		Name:            spec.Name,
		Provider:        spec.Provider,
		TargetNamespace: spec.TargetNamespace,
		Phase:           "staged",
		Message:         fmt.Sprintf("%s infrastructure request staged in GitHub and awaiting controller execution.", strings.Title(spec.Provider)),
		Executed:        false,
		ExecutionState:  "staged",
		Inputs:          spec.Inputs,
		RepoURL:         gitCtx.RepoHTTPSURL,
		Revision:        gitCtx.Branch,
		Path:            gitCtx.Path,
		ApplicationName: gitCtx.ApplicationName,
		ExecutionPlan:   newInfrastructureExecutionPlan("infrastructure_apply_"+spec.Provider, spec),
	}

	var files map[string]string
	switch spec.Provider {
	case "terraform":
		files = renderTerraformInfraFiles(spec)
		record.Message = fmt.Sprintf("Terraform infrastructure request for %s executed through Argo CD.", spec.TargetNamespace)
	case "crossplane":
		files = renderCrossplaneInfraFiles(spec)
		record.Message = fmt.Sprintf("Crossplane infrastructure request for %s staged in GitHub. Crossplane controllers must be available before execution.", spec.TargetNamespace)
		record.ExecutionState = "awaiting_crossplane_reconciliation"
	default:
		return nil, fmt.Errorf("unsupported infrastructure provider %s", spec.Provider)
	}

	if err := writeFiles(manifestPath, files); err != nil {
		return nil, err
	}

	if err := o.commitAndPush(ctx, gitCtx, fmt.Sprintf("Axiom AI infra %s via %s", spec.Name, spec.Provider)); err != nil {
		return nil, err
	}

	if spec.Provider == "terraform" {
		if err := o.applyArgoCDApplication(ctx, gitCtx, "axiom-infra-jobs"); err != nil {
			return nil, err
		}
		if err := o.waitForApplication(ctx, gitCtx.ApplicationName); err != nil {
			return nil, err
		}
		status, err := o.TerraformInfrastructureStatus(ctx, spec.Name)
		if err != nil {
			return nil, err
		}
		status.RepoURL = gitCtx.RepoHTTPSURL
		status.Revision = gitCtx.Branch
		status.Path = gitCtx.Path
		status.ApplicationName = gitCtx.ApplicationName
		status.ExecutionPlan = newInfrastructureExecutionPlan("infrastructure_apply_terraform", spec)
		status.ExecutionState = status.Phase
		status.Inputs = spec.Inputs
		return status, nil
	}

	if err := o.applyArgoCDApplication(ctx, gitCtx, spec.TargetNamespace); err != nil {
		return nil, err
	}
	appStatus, _ := o.getApplicationStatus(ctx, gitCtx.ApplicationName)
	if appStatus != nil {
		record.SyncStatus = appStatus.SyncStatus
		record.HealthStatus = appStatus.HealthStatus
		record.ExecutionState = strings.ToLower(strings.TrimSpace(appStatus.SyncStatus))
		if record.ExecutionState == "" {
			record.ExecutionState = "staged"
		}
	}
	return record, nil
}

func (o *gitOpsOrchestrator) TerraformInfrastructureStatus(ctx context.Context, name string) (*infrastructureRecord, error) {
	jobName := fmt.Sprintf("tf-apply-%s", normalizeKubernetesName(name))
	output, err := runCommand(ctx, "", "kubectl", "-n", "axiom-infra-jobs", "get", "job", jobName, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch terraform job status: %w", err)
	}

	var job struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Succeeded int32 `json:"succeeded"`
			Failed    int32 `json:"failed"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &job); err != nil {
		return nil, fmt.Errorf("failed to parse terraform job status: %w", err)
	}

	appStatus, _ := o.getApplicationStatus(ctx, "infra-"+normalizeKubernetesName(name))
	record := &infrastructureRecord{
		Name:            name,
		Provider:        "terraform",
		TargetNamespace: normalizeKubernetesName(name),
		Executed:        job.Status.Succeeded > 0,
		ApplicationName: "infra-" + normalizeKubernetesName(name),
		Outputs: map[string]string{
			"job": job.Metadata.Name,
		},
		ExecutionState: func() string {
			switch {
			case job.Status.Succeeded > 0:
				return "ready"
			case job.Status.Failed > 0:
				return "failed"
			default:
				return "progressing"
			}
		}(),
	}
	switch {
	case job.Status.Succeeded > 0:
		record.Phase = "ready"
		record.Message = fmt.Sprintf("Terraform created Kubernetes infrastructure for %s.", name)
	case job.Status.Failed > 0:
		record.Phase = "failed"
		record.Message = fmt.Sprintf("Terraform job for %s failed.", name)
	default:
		record.Phase = "progressing"
		record.Message = fmt.Sprintf("Terraform job for %s is still running.", name)
	}
	if appStatus != nil {
		record.SyncStatus = appStatus.SyncStatus
		record.HealthStatus = appStatus.HealthStatus
	}
	record.ExecutionPlan = newInfrastructureExecutionPlan("infrastructure_status_terraform", infrastructureApplyRequest{
		Name:            name,
		Provider:        "terraform",
		TargetNamespace: normalizeKubernetesName(name),
		Inputs:          map[string]string{},
	})
	return record, nil
}

type applicationStatus struct {
	Name         string
	SyncStatus   string
	HealthStatus string
}

func (o *gitOpsOrchestrator) prepareGitOpsBranch(ctx context.Context, category, name string) (*gitOpsContext, error) {
	repoURL := strings.TrimSpace(o.cfg.GitOpsRepoURL)
	if repoURL == "" {
		out, err := runCommand(ctx, "", "git", "config", "--get", "remote.origin.url")
		if err != nil {
			return nil, fmt.Errorf("failed to determine gitops repo url: %w", err)
		}
		repoURL = strings.TrimSpace(string(out))
	}

	repoHTTPS := normalizeGitHubRepoURL(repoURL)
	baseBranch := strings.TrimSpace(o.cfg.GitOpsBaseBranch)
	if baseBranch == "" {
		baseBranch = "main"
	}
	name = normalizeKubernetesName(name)
	branch := fmt.Sprintf("axiom-ai/%s-%s-%s", category, name, time.Now().UTC().Format("20060102150405"))
	basePath := strings.Trim(strings.TrimSpace(o.cfg.GitOpsBasePath), "/")
	if basePath == "" {
		basePath = "deployments/ai-delivery"
	}
	manifestPath := filepath.ToSlash(filepath.Join(basePath, category+"-"+name))
	localDir, err := os.MkdirTemp("", "axiom-gitops-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create gitops temp dir: %w", err)
	}

	if _, err := runCommand(ctx, "", "git", "clone", repoURL, localDir); err != nil {
		return nil, fmt.Errorf("failed to clone gitops repo: %w", err)
	}
	if _, err := runCommand(ctx, localDir, "git", "checkout", baseBranch); err != nil {
		return nil, fmt.Errorf("failed to checkout base branch %s: %w", baseBranch, err)
	}
	if _, err := runCommand(ctx, localDir, "git", "checkout", "-b", branch); err != nil {
		return nil, fmt.Errorf("failed to create gitops branch %s: %w", branch, err)
	}

	return &gitOpsContext{
		RepoURL:         repoURL,
		RepoHTTPSURL:    repoHTTPS,
		BaseBranch:      baseBranch,
		Branch:          branch,
		Path:            manifestPath,
		LocalDir:        localDir,
		ApplicationName: buildApplicationName(category, name),
	}, nil
}

func (o *gitOpsOrchestrator) commitAndPush(ctx context.Context, gitCtx *gitOpsContext, message string) error {
	if _, err := runCommand(ctx, gitCtx.LocalDir, "git", "config", "user.name", fallbackString(o.cfg.GitAuthorName, "axiom-bot")); err != nil {
		return err
	}
	if _, err := runCommand(ctx, gitCtx.LocalDir, "git", "config", "user.email", fallbackString(o.cfg.GitAuthorEmail, "axiom-bot@users.noreply.github.com")); err != nil {
		return err
	}
	if _, err := runCommand(ctx, gitCtx.LocalDir, "git", "add", gitCtx.Path); err != nil {
		return fmt.Errorf("failed to stage gitops manifests: %w", err)
	}
	if _, err := runCommand(ctx, gitCtx.LocalDir, "git", "commit", "-m", message); err != nil {
		return fmt.Errorf("failed to commit gitops manifests: %w", err)
	}
	if _, err := runCommand(ctx, gitCtx.LocalDir, "git", "push", "-u", "origin", gitCtx.Branch); err != nil {
		return fmt.Errorf("failed to push gitops branch: %w", err)
	}
	return nil
}

func (o *gitOpsOrchestrator) applyArgoCDApplication(ctx context.Context, gitCtx *gitOpsContext, destinationNamespace string) error {
	argoNS := fallbackString(o.cfg.ArgoCDNamespace, "argocd")
	project := fallbackString(o.cfg.ArgoCDProject, "default")
	manifest := fmt.Sprintf(`apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: %s
  namespace: %s
spec:
  project: %s
  source:
    repoURL: %s
    targetRevision: %s
    path: %s
  destination:
    server: https://kubernetes.default.svc
    namespace: %s
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
`, gitCtx.ApplicationName, argoNS, project, gitCtx.RepoHTTPSURL, gitCtx.Branch, gitCtx.Path, destinationNamespace)

	cmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to apply argocd application: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func (o *gitOpsOrchestrator) waitForApplication(ctx context.Context, applicationName string) error {
	argoNS := fallbackString(o.cfg.ArgoCDNamespace, "argocd")
	waitCtx, cancel := context.WithTimeout(ctx, fallbackDuration(o.cfg.KubernetesRolloutTimeout, 180*time.Second))
	defer cancel()
	if _, err := runCommand(waitCtx, "", "kubectl", "-n", argoNS, "wait", "--for=jsonpath={.status.sync.status}=Synced", "application/"+applicationName, "--timeout="+fallbackDuration(o.cfg.KubernetesRolloutTimeout, 180*time.Second).String()); err != nil {
		return fmt.Errorf("argocd sync did not complete: %w", err)
	}
	if _, err := runCommand(waitCtx, "", "kubectl", "-n", argoNS, "wait", "--for=jsonpath={.status.health.status}=Healthy", "application/"+applicationName, "--timeout="+fallbackDuration(o.cfg.KubernetesRolloutTimeout, 180*time.Second).String()); err != nil {
		return fmt.Errorf("argocd application did not become healthy: %w", err)
	}
	return nil
}

func (o *gitOpsOrchestrator) getApplicationStatus(ctx context.Context, applicationName string) (*applicationStatus, error) {
	argoNS := fallbackString(o.cfg.ArgoCDNamespace, "argocd")
	output, err := runCommand(ctx, "", "kubectl", "-n", argoNS, "get", "application", applicationName, "-o", "json")
	if err != nil {
		return nil, err
	}
	var app struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			Sync struct {
				Status string `json:"status"`
			} `json:"sync"`
			Health struct {
				Status string `json:"status"`
			} `json:"health"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &app); err != nil {
		return nil, err
	}
	return &applicationStatus{
		Name:         app.Metadata.Name,
		SyncStatus:   app.Status.Sync.Status,
		HealthStatus: app.Status.Health.Status,
	}, nil
}

func (o *gitOpsOrchestrator) getKubernetesDeploymentStatus(ctx context.Context, namespace, name string) (*deploymentRecord, error) {
	namespace = normalizeKubernetesName(namespace)
	name = normalizeKubernetesName(name)
	output, err := runCommand(ctx, "", "kubectl", "-n", namespace, "get", "deployment", name, "-o", "json")
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
				Message string `json:"message"`
			} `json:"conditions"`
		} `json:"status"`
	}
	if err := json.Unmarshal(output, &deployment); err != nil {
		return nil, err
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
		if record.Message == "" && condition.Message != "" {
			record.Message = condition.Message
		}
	}
	serviceOutput, err := runCommand(ctx, "", "kubectl", "-n", namespace, "get", "service", name, "-o", "json")
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
	if record.Message == "" {
		record.Message = fmt.Sprintf("%s has %d/%d ready replicas", record.Name, record.ReadyReplicas, record.Replicas)
	}
	return record, nil
}

func renderDeploymentOnlyManifest(req deploymentApplyRequest) string {
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
`, req.Name, req.Namespace, req.Name, req.Replicas, req.Name, req.Name, req.Name, req.Image, req.Port)
}

func renderServiceOnlyManifest(req deploymentApplyRequest) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: %s
  namespace: %s
  labels:
    app.kubernetes.io/name: %s
spec:
  type: %s
  selector:
    app.kubernetes.io/name: %s
  ports:
    - name: http
      port: %d
      targetPort: %d
`, req.Name, req.Namespace, req.Name, req.ServiceType, req.Name, req.Port, req.Port)
}

func normalizeInfrastructureRequest(req infrastructureApplyRequest) (infrastructureApplyRequest, error) {
	req.Provider = strings.ToLower(strings.TrimSpace(req.Provider))
	req.Name = normalizeKubernetesName(req.Name)
	req.TargetNamespace = normalizeKubernetesName(req.TargetNamespace)
	if req.Name == "" {
		return infrastructureApplyRequest{}, errors.New("infrastructure name is required")
	}
	if req.TargetNamespace == "" {
		req.TargetNamespace = req.Name
	}
	switch req.Provider {
	case "terraform", "crossplane":
	default:
		return infrastructureApplyRequest{}, errors.New("infrastructure provider must be terraform or crossplane")
	}
	if req.Inputs == nil {
		req.Inputs = map[string]string{}
	}
	return req, nil
}

func renderTerraformInfraFiles(req infrastructureApplyRequest) map[string]string {
	jobName := fmt.Sprintf("tf-apply-%s", req.Name)
	return map[string]string{
		"namespace.yaml":        "apiVersion: v1\nkind: Namespace\nmetadata:\n  name: axiom-infra-jobs\n",
		"target-namespace.yaml": fmt.Sprintf("apiVersion: v1\nkind: Namespace\nmetadata:\n  name: %s\n  labels:\n    managed-by: argocd\n    axiom-idp: \"true\"\n", req.TargetNamespace),
		"serviceaccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  name: terraform-runner
  namespace: axiom-infra-jobs
`,
		"clusterrole.yaml": `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: terraform-runner
rules:
  - apiGroups: [""]
    resources: ["namespaces","configmaps","resourcequotas"]
    verbs: ["get","list","watch","create","update","patch"]
`,
		"clusterrolebinding.yaml": `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: terraform-runner
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: terraform-runner
subjects:
  - kind: ServiceAccount
    name: terraform-runner
    namespace: axiom-infra-jobs
`,
		"configmap.yaml": fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s-config
  namespace: axiom-infra-jobs
data:
  main.tf: |
    terraform {
      required_version = ">= 1.6.0"
      required_providers {
        kubernetes = {
          source  = "hashicorp/kubernetes"
          version = "~> 2.31"
        }
      }
    }
    provider "kubernetes" {
      host                   = "https://kubernetes.default.svc"
      token                  = file("/var/run/secrets/kubernetes.io/serviceaccount/token")
      cluster_ca_certificate = file("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
    }
    resource "kubernetes_config_map" "footprint" {
      metadata {
        name      = "terraform-footprint"
        namespace = "%s"
      }
      data = {
        request = "%s"
        provider = "terraform"
      }
    }
    resource "kubernetes_resource_quota_v1" "guardrails" {
      metadata {
        name      = "terraform-guardrails"
        namespace = "%s"
      }
      spec {
        hard = {
          "pods" = "10"
          "services" = "5"
        }
      }
    }
`, jobName, req.TargetNamespace, req.Name, req.TargetNamespace),
		"job.yaml": fmt.Sprintf(`apiVersion: batch/v1
kind: Job
metadata:
  name: %s
  namespace: axiom-infra-jobs
spec:
  backoffLimit: 1
  template:
    spec:
      serviceAccountName: terraform-runner
      restartPolicy: Never
      containers:
        - name: terraform
          image: hashicorp/terraform:1.8.5
          command:
            - sh
            - -lc
            - |
              mkdir -p /tmp/work
              cp /workspace/main.tf /tmp/work/main.tf
              cd /tmp/work
              terraform init
              terraform apply -auto-approve
          volumeMounts:
            - name: workspace
              mountPath: /workspace
      volumes:
        - name: workspace
          configMap:
            name: %s-config
`, jobName, jobName),
		"kustomization.yaml": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- namespace.yaml\n- target-namespace.yaml\n- serviceaccount.yaml\n- clusterrole.yaml\n- clusterrolebinding.yaml\n- configmap.yaml\n- job.yaml\n",
	}
}

func renderCrossplaneInfraFiles(req infrastructureApplyRequest) map[string]string {
	inputs := renderJSON(req.Inputs)
	return map[string]string{
		"README.md": fmt.Sprintf(`# Crossplane Infrastructure Request

This request was generated by Axiom IDP AI.

- Name: %s
- Target namespace: %s
- Provider: crossplane
- Inputs: %s

Execution requires Crossplane and an installed provider/configuration in the target cluster.
`, req.Name, req.TargetNamespace, inputs),
		"claim.yaml": fmt.Sprintf(`apiVersion: platform.axiom.dev/v1alpha1
kind: %sClaim
metadata:
  name: %s
  namespace: %s
spec:
  targetNamespace: %s
  provider: crossplane
  parameters:
%s
`, strings.Title(req.Name), req.Name, req.TargetNamespace, req.TargetNamespace, indentYAMLMap(req.Inputs, 4)),
		"kustomization.yaml": "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\nresources:\n- claim.yaml\n",
	}
}

func indentYAMLMap(values map[string]string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	if len(values) == 0 {
		return indent + "{}"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(values))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s%s: %q", indent, key, values[key]))
	}
	return strings.Join(lines, "\n")
}

func writeFiles(dir string, files map[string]string) error {
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			return fmt.Errorf("failed to write %s: %w", name, err)
		}
	}
	return nil
}

func runCommand(ctx context.Context, dir string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return output, nil
}

func normalizeGitHubRepoURL(url string) string {
	url = strings.TrimSpace(url)
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
		return "https://github.com/" + url + ".git"
	}
	if strings.HasPrefix(url, "https://github.com/") && !strings.HasSuffix(url, ".git") {
		return url + ".git"
	}
	return url
}

func buildApplicationName(category, name string) string {
	if category == "terraform" || category == "crossplane" {
		return "infra-" + normalizeKubernetesName(name)
	}
	return normalizeKubernetesName(name)
}

func fallbackString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func fallbackDuration(value, fallback time.Duration) time.Duration {
	if value <= 0 {
		return fallback
	}
	return value
}

var (
	infraProviderPattern = regexp.MustCompile(`\b(terraform|crossplane)\b`)
	infraNamePattern     = regexp.MustCompile(`(?:infra(?:structure)?|stack|platform|environment)\s+(?:named\s+|called\s+)?([a-z0-9][a-z0-9-]{1,41})`)
	infraTargetPattern   = regexp.MustCompile(`(?:namespace|environment)\s+([a-z0-9][a-z0-9-]{1,41})`)
)

func parseInfrastructurePrompt(query string) (infrastructureApplyRequest, bool) {
	lower := strings.ToLower(strings.TrimSpace(query))
	if lower == "" || !containsAny(lower, "infra", "infrastructure", "provision", "platform", "environment", "namespace") {
		return infrastructureApplyRequest{}, false
	}
	providerMatch := infraProviderPattern.FindStringSubmatch(lower)
	if len(providerMatch) != 2 {
		return infrastructureApplyRequest{}, false
	}
	req := infrastructureApplyRequest{
		Provider: providerMatch[1],
		Inputs:   map[string]string{},
	}
	if match := infraTargetPattern.FindStringSubmatch(lower); len(match) == 2 {
		req.Name = match[1]
		req.TargetNamespace = match[1]
	}
	if match := infraNamePattern.FindStringSubmatch(lower); len(match) == 2 {
		if req.Name == "" {
			req.Name = match[1]
		}
		if req.TargetNamespace == "" {
			req.TargetNamespace = match[1]
		}
	}
	if req.Name == "" {
		req.Name = providerMatch[1] + "-request"
		req.TargetNamespace = req.Name
	}
	if match := namespacePattern.FindStringSubmatch(lower); len(match) == 2 {
		req.TargetNamespace = match[1]
	}
	return req, true
}

func parseArgoCDDeploymentPrompt(query, defaultNamespace string) (deploymentApplyRequest, bool) {
	req, ok := parseDeploymentPrompt(query, defaultNamespace)
	if !ok {
		return deploymentApplyRequest{}, false
	}
	if containsAny(strings.ToLower(query), "argocd", "argo cd", "github") {
		req.Delivery = "argocd"
		return req, true
	}
	return deploymentApplyRequest{}, false
}

func parseArgoCDStatusPrompt(query, defaultNamespace string) (string, string, bool) {
	if !containsAny(strings.ToLower(query), "argocd", "argo cd", "github") {
		return "", "", false
	}
	return parseDeploymentStatusPrompt(query, defaultNamespace)
}

func renderJSON(v interface{}) string {
	buf := &bytes.Buffer{}
	_ = json.NewEncoder(buf).Encode(v)
	return strings.TrimSpace(buf.String())
}

func safeAtoi(value string, fallback int) int {
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed
	}
	return fallback
}
