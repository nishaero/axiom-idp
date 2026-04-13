package utils

import (
	"testing"
	"strings"
)

func TestGenerateID(t *testing.T) {
	id := GenerateID("test")

	if !strings.HasPrefix(id, "test-") {
		t.Errorf("ID should start with prefix, got %s", id)
	}

	if len(id) < 10 {
		t.Errorf("ID should be long enough, got %s", id)
	}

	// Test uniqueness
	id2 := GenerateID("test")
	if id == id2 {
		t.Error("IDs should be unique")
	}
}

func TestHash(t *testing.T) {
	input := "test-string"
	hash := Hash(input)

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	if len(hash) != 64 {
		t.Errorf("SHA256 hash should be 64 chars, got %d", len(hash))
	}

	// Test consistency
	hash2 := Hash(input)
	if hash != hash2 {
		t.Error("Same input should produce same hash")
	}

	// Test different input
	hash3 := Hash("different")
	if hash == hash3 {
		t.Error("Different inputs should produce different hashes")
	}
}
