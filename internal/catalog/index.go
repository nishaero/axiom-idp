package catalog

import (
	"strings"
	"sync"
)

// Service represents a service in the catalog
type Service struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Owner       string                 `json:"owner"`
	Tags        []string               `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Index maintains the service catalog
type Index struct {
	services map[string]*Service
	mu       sync.RWMutex
}

// NewIndex creates a new catalog index
func NewIndex() *Index {
	return &Index{
		services: make(map[string]*Service),
	}
}

// Add adds a service to the catalog
func (i *Index) Add(service *Service) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.services[service.ID]; exists {
		return ErrServiceAlreadyExists
	}

	i.services[service.ID] = service
	return nil
}

// Get retrieves a service by ID
func (i *Index) Get(id string) (*Service, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	service, exists := i.services[id]
	if !exists {
		return nil, ErrServiceNotFound
	}

	return service, nil
}

// List returns all services
func (i *Index) List() []*Service {
	i.mu.RLock()
	defer i.mu.RUnlock()

	services := make([]*Service, 0, len(i.services))
	for _, service := range i.services {
		services = append(services, service)
	}

	return services
}

// Search searches for services by tag or name
func (i *Index) Search(query string) []*Service {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var results []*Service
	for _, service := range i.services {
		if matchesQuery(service, query) {
			results = append(results, service)
		}
	}

	return results
}

// Delete removes a service from the catalog
func (i *Index) Delete(id string) error {
	i.mu.Lock()
	defer i.mu.Unlock()

	if _, exists := i.services[id]; !exists {
		return ErrServiceNotFound
	}

	delete(i.services, id)
	return nil
}

// Count returns the number of services in the catalog
func (i *Index) Count() int {
	i.mu.RLock()
	defer i.mu.RUnlock()

	return len(i.services)
}

// Helper function for matching
func matchesQuery(service *Service, query string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}

	if strings.Contains(strings.ToLower(service.Name), query) || strings.Contains(strings.ToLower(service.Description), query) {
		return true
	}

	for _, tag := range service.Tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}

	for _, value := range service.Metadata {
		if strings.Contains(strings.ToLower(strings.TrimSpace(strings.TrimSpace(toString(value)))), query) {
			return true
		}
	}

	return false
}

func toString(v interface{}) string {
	switch t := v.(type) {
	case string:
		return t
	case []string:
		return strings.Join(t, " ")
	default:
		return ""
	}
}
