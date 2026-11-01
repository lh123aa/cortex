package models

import "testing"

func TestNormalizeRole(t *testing.T) {
	cases := map[string]string{
		"admin":   RoleAdmin,
		"ADMIN":   RoleAdmin,
		" editor": RoleEditor,
		"viewer":  RoleViewer,
		"user":    RoleEditor, // 历史值等价于 editor
		"":        RoleEditor, // 空值沿用旧版可写行为
		"banana":  RoleViewer, // 未知值按最小权限
	}
	for in, want := range cases {
		if got := NormalizeRole(in); got != want {
			t.Errorf("NormalizeRole(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	for _, r := range []string{RoleAdmin, RoleEditor, RoleViewer, RoleUser} {
		if !IsValidRole(r) {
			t.Errorf("%q should be valid", r)
		}
	}
	for _, r := range []string{"", "root", "superuser", "banana"} {
		if IsValidRole(r) {
			t.Errorf("%q should be invalid", r)
		}
	}
}

func TestRoleRankOrdering(t *testing.T) {
	if RoleRank(RoleViewer) >= RoleRank(RoleEditor) {
		t.Error("viewer must rank below editor")
	}
	if RoleRank(RoleEditor) >= RoleRank(RoleAdmin) {
		t.Error("editor must rank below admin")
	}
	if RoleRank(RoleUser) != RoleRank(RoleEditor) {
		t.Error("legacy user role must rank as editor")
	}
}

func TestPermissionMatrix(t *testing.T) {
	cases := []struct {
		role     string
		canRead  bool
		canWrite bool
		canAdmin bool
	}{
		{RoleViewer, true, false, false},
		{RoleEditor, true, true, false},
		{RoleAdmin, true, true, true},
		{RoleUser, true, true, false},  // 历史值
		{"", true, true, false},        // 空值按 editor
		{"banana", true, false, false}, // 未知值按 viewer
	}
	for _, c := range cases {
		if got := CanRead(c.role); got != c.canRead {
			t.Errorf("CanRead(%q) = %v, want %v", c.role, got, c.canRead)
		}
		if got := CanWrite(c.role); got != c.canWrite {
			t.Errorf("CanWrite(%q) = %v, want %v", c.role, got, c.canWrite)
		}
		if got := CanAdmin(c.role); got != c.canAdmin {
			t.Errorf("CanAdmin(%q) = %v, want %v", c.role, got, c.canAdmin)
		}
	}
}

// TestRolePrivilegeIsMonotonic 权限必须单调：高角色拥有的能力，低角色不能多。
func TestRolePrivilegeIsMonotonic(t *testing.T) {
	if CanAdmin(RoleViewer) || CanAdmin(RoleEditor) {
		t.Error("only admin should have admin rights")
	}
	if CanWrite(RoleViewer) {
		t.Error("viewer must not have write rights")
	}
}
