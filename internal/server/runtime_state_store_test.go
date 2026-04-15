package server

import (
	"context"
	"testing"
	"time"

	"github.com/axiom-idp/axiom/internal/config"
)

func TestSQLRuntimeStateStorePersistsAuditAndRateLimit(t *testing.T) {
	cfg := &config.Config{
		DBDriver: "sqlite3",
		DBURL:    "file:axiom-state-test?mode=memory&cache=shared",
	}

	store, err := newRuntimeStateStore(cfg)
	if err != nil {
		t.Fatalf("Failed to create runtime state store: %v", err)
	}
	if store == nil {
		t.Fatal("Expected SQL runtime state store")
	}
	defer store.Close()

	entry := AuditLog{
		ID:        "audit-1",
		UserID:    "user-1",
		Action:    "deploy",
		Resource:  "catalog",
		Status:    "success",
		Details:   map[string]interface{}{"service": "demo"},
		CreatedAt: time.Now().UTC(),
	}
	if err := store.AppendAudit(context.Background(), entry); err != nil {
		t.Fatalf("Failed to append audit entry: %v", err)
	}

	logs, err := store.QueryAudit(context.Background(), "user-1", 10)
	if err != nil {
		t.Fatalf("Failed to query audit entries: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Expected 1 audit entry, got %d", len(logs))
	}
	if logs[0].Details["service"] != "demo" {
		t.Fatalf("Expected stored audit details, got %v", logs[0].Details)
	}

	stats, err := store.AuditStats(context.Background())
	if err != nil {
		t.Fatalf("Failed to collect audit stats: %v", err)
	}
	if stats.Entries != 1 || stats.SuccessCount != 1 {
		t.Fatalf("Expected audit stats to reflect persisted entry, got %+v", stats)
	}

	allowed, err := store.AllowRateLimit(context.Background(), "user:user-1", 2, time.Minute)
	if err != nil {
		t.Fatalf("Failed to create first rate limit record: %v", err)
	}
	if !allowed {
		t.Fatal("Expected first rate limit check to pass")
	}

	allowed, err = store.AllowRateLimit(context.Background(), "user:user-1", 2, time.Minute)
	if err != nil {
		t.Fatalf("Failed to create second rate limit record: %v", err)
	}
	if !allowed {
		t.Fatal("Expected second rate limit check to pass")
	}

	allowed, err = store.AllowRateLimit(context.Background(), "user:user-1", 2, time.Minute)
	if err != nil {
		t.Fatalf("Failed to create third rate limit record: %v", err)
	}
	if allowed {
		t.Fatal("Expected third rate limit check to be denied")
	}

	rateStats, err := store.RateLimitStats(context.Background())
	if err != nil {
		t.Fatalf("Failed to collect rate limit stats: %v", err)
	}
	if rateStats.TrackedKeys != 1 || rateStats.RequestsSeen != 3 {
		t.Fatalf("Expected rate limit stats to reflect persisted windows, got %+v", rateStats)
	}
	if store.Shared() {
		t.Fatal("Expected sqlite-backed store to remain local")
	}
	if store.Backend() != "sqlite" {
		t.Fatalf("Expected sqlite backend, got %q", store.Backend())
	}
}
