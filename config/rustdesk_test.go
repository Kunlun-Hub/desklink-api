package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadKeyFileOverridesBundledKey(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "id_ed25519.pub")
	if err := os.WriteFile(path, []byte("mounted-server-key"), 0o600); err != nil {
		t.Fatal(err)
	}
	config := &Rustdesk{Key: "bundled-development-key", KeyFile: path}
	config.LoadKeyFile()
	if config.Key != "mounted-server-key" {
		t.Fatalf("key = %q, want mounted server key", config.Key)
	}
}

func TestLoadKeyFileKeepsExplicitKeyWhenFileMissing(t *testing.T) {
	config := &Rustdesk{Key: "explicit-key", KeyFile: "/does/not/exist"}
	config.LoadKeyFile()
	if config.Key != "explicit-key" {
		t.Fatalf("key = %q, want explicit key", config.Key)
	}
}
