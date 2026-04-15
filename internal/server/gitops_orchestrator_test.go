package server

import (
	"context"
	"os"
	"testing"

	"github.com/axiom-idp/axiom/internal/config"
)

func TestResolveGitOpsRepoURLFromConfig(t *testing.T) {
	t.Setenv("AXIOM_GITOPS_REPO_URL", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_SERVER_URL", "")

	orchestrator := newGitOpsOrchestrator(&config.Config{
		GitOpsRepoURL: "https://github.com/example/platform-config.git",
	}, nil)

	repoURL, err := orchestrator.resolveGitOpsRepoURL(context.Background())
	if err != nil {
		t.Fatalf("resolveGitOpsRepoURL returned error: %v", err)
	}
	if repoURL != "https://github.com/example/platform-config.git" {
		t.Fatalf("expected config repo url, got %q", repoURL)
	}
}

func TestResolveGitOpsRepoURLFromEnvironment(t *testing.T) {
	t.Setenv("AXIOM_GITOPS_REPO_URL", "https://github.com/example/runtime-config.git")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_SERVER_URL", "")

	orchestrator := newGitOpsOrchestrator(&config.Config{}, nil)

	repoURL, err := orchestrator.resolveGitOpsRepoURL(context.Background())
	if err != nil {
		t.Fatalf("resolveGitOpsRepoURL returned error: %v", err)
	}
	if repoURL != "https://github.com/example/runtime-config.git" {
		t.Fatalf("expected runtime env repo url, got %q", repoURL)
	}
}

func TestResolveGitOpsRepoURLFromGitHubRepositoryEnvironment(t *testing.T) {
	t.Setenv("AXIOM_GITOPS_REPO_URL", "")
	t.Setenv("GITHUB_REPOSITORY", "acme/axiom-config")
	t.Setenv("GITHUB_SERVER_URL", "https://github.example.com")

	orchestrator := newGitOpsOrchestrator(&config.Config{}, nil)

	repoURL, err := orchestrator.resolveGitOpsRepoURL(context.Background())
	if err != nil {
		t.Fatalf("resolveGitOpsRepoURL returned error: %v", err)
	}
	if repoURL != "https://github.example.com/acme/axiom-config.git" {
		t.Fatalf("expected GitHub repository env repo url, got %q", repoURL)
	}
}

func TestResolveGitOpsRepoURLRequiresFallbackSource(t *testing.T) {
	t.Setenv("AXIOM_GITOPS_REPO_URL", "")
	t.Setenv("GITHUB_REPOSITORY", "")
	t.Setenv("GITHUB_SERVER_URL", "")

	previousPath := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer func() {
		_ = os.Setenv("PATH", previousPath)
	}()

	orchestrator := newGitOpsOrchestrator(&config.Config{}, nil)

	if _, err := orchestrator.resolveGitOpsRepoURL(context.Background()); err == nil {
		t.Fatal("expected resolveGitOpsRepoURL to fail without config, env, or git")
	}
}
