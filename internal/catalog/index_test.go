package catalog

import (
	"testing"
)

func TestAdd(t *testing.T) {
	index := NewIndex()

	service := &Service{
		ID:          "test-1",
		Name:        "Test Service",
		Description: "A test service",
		Owner:       "test-owner",
		Tags:        []string{"web", "api"},
	}

	err := index.Add(service)
	if err != nil {
		t.Fatalf("Failed to add service: %v", err)
	}

	if index.Count() != 1 {
		t.Errorf("Expected 1 service, got %d", index.Count())
	}
}

func TestAddDuplicate(t *testing.T) {
	index := NewIndex()

	service := &Service{
		ID:   "test-1",
		Name: "Test Service",
	}

	index.Add(service)

	err := index.Add(service)
	if err != ErrServiceAlreadyExists {
		t.Errorf("Expected ErrServiceAlreadyExists, got %v", err)
	}
}

func TestGet(t *testing.T) {
	index := NewIndex()

	service := &Service{
		ID:   "test-1",
		Name: "Test Service",
	}

	index.Add(service)

	retrieved, err := index.Get("test-1")
	if err != nil {
		t.Fatalf("Failed to get service: %v", err)
	}

	if retrieved.Name != "Test Service" {
		t.Errorf("Expected name 'Test Service', got '%s'", retrieved.Name)
	}
}

func TestGetNotFound(t *testing.T) {
	index := NewIndex()

	_, err := index.Get("nonexistent")
	if err != ErrServiceNotFound {
		t.Errorf("Expected ErrServiceNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	index := NewIndex()

	for i := 1; i <= 3; i++ {
		service := &Service{
			ID:   "test-" + string(rune(i)),
			Name: "Test Service " + string(rune(i)),
		}
		index.Add(service)
	}

	services := index.List()
	if len(services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(services))
	}
}

func TestDelete(t *testing.T) {
	index := NewIndex()

	service := &Service{
		ID:   "test-1",
		Name: "Test Service",
	}

	index.Add(service)

	err := index.Delete("test-1")
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}

	if index.Count() != 0 {
		t.Errorf("Expected 0 services after delete, got %d", index.Count())
	}
}
