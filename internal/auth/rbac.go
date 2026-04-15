package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

// Role-based access control.

const (
	RoleAdmin       = "admin"
	RoleEngineer    = "engineer"
	RoleViewer      = "viewer"
	RoleContributor = "contributor"
)

// Permission represents an action permission.
type Permission struct {
	Resource string
	Action   string // read, write, delete, admin
}

// RolePermissions defines permissions for a role.
var RolePermissions = map[string][]Permission{
	RoleAdmin: {
		{Resource: "*", Action: "*"},
	},
	RoleEngineer: {
		{Resource: "catalog", Action: "read"},
		{Resource: "catalog", Action: "write"},
		{Resource: "services", Action: "read"},
		{Resource: "services", Action: "deploy"},
	},
	RoleContributor: {
		{Resource: "catalog", Action: "read"},
		{Resource: "services", Action: "read"},
	},
	RoleViewer: {
		{Resource: "catalog", Action: "read"},
		{Resource: "services", Action: "read"},
	},
}

type contextKey string

const (
	userIDContextKey contextKey = "user_id"
	rolesContextKey  contextKey = "roles"
)

// RBAC represents role-based access control.
type RBAC struct {
	mu        sync.RWMutex
	userRoles map[string][]string
}

// NewRBAC creates a new RBAC system.
func NewRBAC() *RBAC {
	return &RBAC{
		userRoles: make(map[string][]string),
	}
}

// AssignRole assigns a role to a user.
func (r *RBAC) AssignRole(userID, role string) error {
	if _, exists := RolePermissions[role]; !exists {
		return fmt.Errorf("invalid role: %s", role)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	roles := r.userRoles[userID]
	for _, existing := range roles {
		if existing == role {
			return nil
		}
	}

	r.userRoles[userID] = append(roles, role)
	return nil
}

// RemoveRole removes a role from a user.
func (r *RBAC) RemoveRole(userID, role string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	roles, exists := r.userRoles[userID]
	if !exists {
		return
	}

	filtered := make([]string, 0, len(roles))
	for _, existing := range roles {
		if existing != role {
			filtered = append(filtered, existing)
		}
	}
	if len(filtered) == 0 {
		delete(r.userRoles, userID)
		return
	}
	r.userRoles[userID] = filtered
}

// RolesForUser returns the assigned roles for a user.
func (r *RBAC) RolesForUser(userID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	roles := r.userRoles[userID]
	if len(roles) == 0 {
		return nil
	}

	out := make([]string, len(roles))
	copy(out, roles)
	return out
}

// CanAccess checks if a user can access a resource.
func (r *RBAC) CanAccess(userID, resource, action string) bool {
	return r.CanAccessRoles(r.RolesForUser(userID), resource, action)
}

// CanAccessRoles checks if the provided roles can access a resource.
func (r *RBAC) CanAccessRoles(roles []string, resource, action string) bool {
	if len(roles) == 0 {
		return false
	}

	for _, role := range roles {
		permissions := RolePermissions[strings.ToLower(strings.TrimSpace(role))]
		for _, perm := range permissions {
			if (perm.Resource == "*" || perm.Resource == resource) &&
				(perm.Action == "*" || perm.Action == action) {
				return true
			}
		}
	}

	return false
}

// Middleware enforces RBAC for a resource/action pair.
func (r *RBAC) Middleware(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userID := UserIDFromContext(req.Context())
			roles := RolesFromContext(req.Context())
			if len(roles) == 0 && userID != "" {
				roles = r.RolesForUser(userID)
			}
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			if !r.CanAccessRoles(roles, resource, action) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// ContextWithUser adds user information to a context.
func ContextWithUser(ctx context.Context, userID string, roles []string) context.Context {
	if userID != "" {
		ctx = context.WithValue(ctx, userIDContextKey, userID)
	}
	if len(roles) > 0 {
		normalized := make([]string, 0, len(roles))
		seen := make(map[string]struct{}, len(roles))
		for _, role := range roles {
			role = strings.TrimSpace(strings.ToLower(role))
			if role == "" {
				continue
			}
			if _, exists := seen[role]; exists {
				continue
			}
			seen[role] = struct{}{}
			normalized = append(normalized, role)
		}
		ctx = context.WithValue(ctx, rolesContextKey, normalized)
	}
	return ctx
}

// ContextWithUserID adds user ID to context.
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return ContextWithUser(ctx, userID, nil)
}

// ContextWithRoles adds roles to context.
func ContextWithRoles(ctx context.Context, roles []string) context.Context {
	if len(roles) == 0 {
		return ctx
	}

	normalized := make([]string, 0, len(roles))
	seen := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		role = strings.TrimSpace(strings.ToLower(role))
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		normalized = append(normalized, role)
	}

	return context.WithValue(ctx, rolesContextKey, normalized)
}

// UserIDFromContext extracts user ID from context.
func UserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value(userIDContextKey).(string)
	return userID
}

// RolesFromContext extracts roles from context.
func RolesFromContext(ctx context.Context) []string {
	roles, _ := ctx.Value(rolesContextKey).([]string)
	if len(roles) == 0 {
		return nil
	}

	out := make([]string, len(roles))
	copy(out, roles)
	return out
}
