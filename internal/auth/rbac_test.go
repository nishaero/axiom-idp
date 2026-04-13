package auth

import (
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
