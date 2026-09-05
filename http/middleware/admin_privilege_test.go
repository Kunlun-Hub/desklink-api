package middleware

import "testing"

func TestPermissionForPath(t *testing.T) {
	tests := map[string]string{
		"/api/admin/peer/list":                    "devices",
		"/api/admin/address_book/list":            "address-books",
		"/api/admin/address_book_collection/list": "collections",
		"/api/admin/audit_conn/list":              "connection-audit",
		"/api/admin/recordings/1/access":          "recordings",
		"/api/admin/rustdesk/sendCmd":             "commands",
		"/api/admin/unknown/list":                 "",
	}
	for path, expected := range tests {
		if actual := permissionForPath(path); actual != expected {
			t.Errorf("permissionForPath(%q) = %q, want %q", path, actual, expected)
		}
	}
}
