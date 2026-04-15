package server

import "strings"

type executionPlan struct {
	Intent    string   `json:"intent"`
	Provider  string   `json:"provider,omitempty"`
	Route     string   `json:"route"`
	Mode      string   `json:"mode"`
	Supported bool     `json:"supported"`
	Notes     []string `json:"notes,omitempty"`
}

func newDeploymentExecutionPlan(intent string, req deploymentApplyRequest) *executionPlan {
	route := "direct-kubernetes"
	mode := "applied"
	notes := []string{
		"Deployment is rendered as Kubernetes manifests before execution.",
	}

	if strings.EqualFold(req.Delivery, "argocd") {
		route = "github-argocd-kubernetes"
		mode = "controller-backed"
		notes = []string{
			"GitHub branch is pushed and Argo CD is applied before rollout checks run.",
		}
	}

	return &executionPlan{
		Intent:    intent,
		Provider:  req.Delivery,
		Route:     route,
		Mode:      mode,
		Supported: true,
		Notes:     notes,
	}
}

func newInfrastructureExecutionPlan(intent string, req infrastructureApplyRequest) *executionPlan {
	route := "github-argocd-terraform-job"
	mode := "controller-backed"
	notes := []string{
		"Terraform infrastructure is staged in GitHub and executed by an in-cluster Job managed through Argo CD.",
	}

	if req.Provider == "crossplane" {
		route = "github-argocd-crossplane-controller"
		mode = "staged"
		notes = []string{
			"Crossplane request bundles are staged in GitHub and synced through Argo CD.",
			"Execution remains controller-dependent until the target cluster has the required XRD, composition, and provider installed.",
		}
	}

	return &executionPlan{
		Intent:    intent,
		Provider:  req.Provider,
		Route:     route,
		Mode:      mode,
		Supported: true,
		Notes:     notes,
	}
}
