package server

import (
	"context"
	"testing"
)

func TestAuditorLog(t *testing.T) {
	auditor := NewAuditor()

	auditor.Log(context.Background(), "user-1", "access", "catalog", "success", map[string]interface{}{})

	logs := auditor.GetLogs("user-1", 10)
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	if logs[0].Action != "access" {
		t.Errorf("Expected action 'access', got '%s'", logs[0].Action)
	}
}

func TestAuditorLogError(t *testing.T) {
	auditor := NewAuditor()

	err := ErrUnauthorized
	auditor.LogError(context.Background(), "user-1", "access", "catalog", err)

	logs := auditor.GetLogs("user-1", 10)
	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	if logs[0].Status != "error" {
		t.Errorf("Expected status 'error', got '%s'", logs[0].Status)
	}
}

func TestAuditorGetLogs(t *testing.T) {
	auditor := NewAuditor()

	auditor.Log(context.Background(), "user-1", "action1", "catalog", "success", map[string]interface{}{})
	auditor.Log(context.Background(), "user-2", "action2", "catalog", "success", map[string]interface{}{})
	auditor.Log(context.Background(), "user-1", "action3", "catalog", "success", map[string]interface{}{})

	logs := auditor.GetLogs("user-1", 10)
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries for user-1, got %d", len(logs))
	}

	allLogs := auditor.GetLogs("", 10)
	if len(allLogs) != 3 {
		t.Errorf("Expected 3 total log entries, got %d", len(allLogs))
	}
}
