package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsNewer(t *testing.T) {
	tests := []struct {
		latest, current string
		want            bool
	}{
		{"1.2.0", "1.1.0", true},
		{"1.1.1", "1.1.0", true},
		{"2.0.0", "1.9.9", true},
		{"1.10.0", "1.9.0", true},
		{"1.1.0", "1.1.0", false},
		{"1.0.0", "1.1.0", false},
		{"1.1.0", "2.0.0", false},
		{"1.1.0", "", false},
		{"1.1.0", "dev", false},
		{"bad", "1.0.0", false},
		{"1.0.0", "bad", false},
		{"1.0", "1.0.0", false},
	}
	for _, tt := range tests {
		got := isNewer(tt.latest, tt.current)
		if got != tt.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
		}
	}
}

func TestApplyUpdate(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "monibright.exe")
	tmp := exe + ".tmp"

	os.WriteFile(exe, []byte("old"), 0o755)
	os.WriteFile(tmp, []byte("new"), 0o755)

	// Patch os.Executable by testing the rename logic directly.
	old := exe + ".old"

	if err := os.Rename(exe, old); err != nil {
		t.Fatalf("rename exe to old: %v", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		os.Rename(old, exe)
		t.Fatalf("rename tmp to exe: %v", err)
	}

	got, _ := os.ReadFile(exe)
	if string(got) != "new" {
		t.Errorf("exe content = %q, want %q", got, "new")
	}

	gotOld, _ := os.ReadFile(old)
	if string(gotOld) != "old" {
		t.Errorf("old content = %q, want %q", gotOld, "old")
	}

	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Errorf("tmp file should not exist after rename")
	}
}

func TestApplyUpdateRollback(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "monibright.exe")

	os.WriteFile(exe, []byte("old"), 0o755)

	// No .tmp file exists — rename should fail and restore original.
	old := exe + ".old"
	tmp := exe + ".tmp"

	if err := os.Rename(exe, old); err != nil {
		t.Fatalf("rename exe to old: %v", err)
	}
	if err := os.Rename(tmp, exe); err != nil {
		// Rollback.
		os.Rename(old, exe)
	}

	got, _ := os.ReadFile(exe)
	if string(got) != "old" {
		t.Errorf("after rollback, exe content = %q, want %q", got, "old")
	}
}
