package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRBACAssignRole(t *testing.T) {
	rbac := NewRBAC()

	err := rbac.AssignRole("user-1", RoleAdmin)
	if err != nil {
		t.Fatalf("Failed to assign role: %v", err)
	}

	err = rbac.AssignRole("user-1", "invalid-role")
	if err == nil {
		t.Error("Should return error for invalid role")
	}
}

func TestRBACCanAccess(t *testing.T) {
	rbac := NewRBAC()

	rbac.AssignRole("user-1", RoleAdmin)

	if !rbac.CanAccess("user-1", "catalog", "read") {
		t.Error("Admin should have access to catalog read")
	}

	rbac.AssignRole("user-2", RoleViewer)

	if !rbac.CanAccess("user-2", "catalog", "read") {
		t.Error("Viewer should have access to catalog read")
	}

	if rbac.CanAccess("user-2", "catalog", "write") {
		t.Error("Viewer should not have access to catalog write")
	}
}

func TestRBACRemoveRole(t *testing.T) {
	rbac := NewRBAC()

	rbac.AssignRole("user-1", RoleAdmin)
	rbac.RemoveRole("user-1", RoleAdmin)

	if rbac.CanAccess("user-1", "catalog", "read") {
		t.Error("User should not have access after role removal")
	}
}

func TestRBACMiddlewareUsesContextRoles(t *testing.T) {
	rbac := NewRBAC()
	handler := rbac.Middleware("catalog", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services", nil)
	req = req.WithContext(ContextWithUser(req.Context(), "user-1", []string{RoleViewer}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}
}

func TestRBACMiddlewareRejectsUnauthorized(t *testing.T) {
	rbac := NewRBAC()
	handler := rbac.Middleware("catalog", "read")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/catalog/services", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected status 401, got %d", w.Code)
	}
}
