package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Role-based access control

const (
	RoleAdmin      = "admin"
	RoleEngineer   = "engineer"
	RoleViewer     = "viewer"
	RoleContributor = "contributor"
)

// Permission represents an action permission
type Permission struct {
	Resource string
	Action   string // read, write, delete, admin
}

// Role defines permissions for a role
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

// RBAC represents role-based access control
type RBAC struct {
	userRoles map[string][]string // userID -> roles
}

// NewRBAC creates a new RBAC system
func NewRBAC() *RBAC {
	return &RBAC{
		userRoles: make(map[string][]string),
	}
}

// AssignRole assigns a role to a user
func (r *RBAC) AssignRole(userID, role string) error {
	if _, exists := RolePermissions[role]; !exists {
		return fmt.Errorf("invalid role: %s", role)
	}

	if _, exists := r.userRoles[userID]; !exists {
		r.userRoles[userID] = []string{}
	}

	r.userRoles[userID] = append(r.userRoles[userID], role)
	return nil
}

// RemoveRole removes a role from a user
func (r *RBAC) RemoveRole(userID, role string) {
	if roles, exists := r.userRoles[userID]; exists {
		filtered := make([]string, 0)
		for _, r := range roles {
			if r != role {
				filtered = append(filtered, r)
			}
		}
		r.userRoles[userID] = filtered
	}
}

// CanAccess checks if a user can access a resource
func (r *RBAC) CanAccess(userID, resource, action string) bool {
	roles, exists := r.userRoles[userID]
	if !exists || len(roles) == 0 {
		return false
	}

	for _, role := range roles {
		permissions := RolePermissions[role]
		for _, perm := range permissions {
			if (perm.Resource == "*" || perm.Resource == resource) &&
				(perm.Action == "*" || perm.Action == action) {
				return true
			}
		}
	}

	return false
}

// Middleware for RBAC enforcement
func (r *RBAC) Middleware(resource, action string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Extract user ID from context
			userID := extractUserID(req)
			if userID == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check access
			if !r.CanAccess(userID, resource, action) {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

// Helper function to extract user ID from request
func extractUserID(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	// TODO: Parse token to extract user ID
	return ""
}

// ContextWithUserID adds user ID to context
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, "user_id", userID)
}

// UserIDFromContext extracts user ID from context
func UserIDFromContext(ctx context.Context) string {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return ""
	}
	return userID
}
