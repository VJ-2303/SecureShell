package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirAndCompare(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	// Create src structure
	if err := os.MkdirAll(filepath.Join(src, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "file1.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "sub", "file2.txt"), []byte("world"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy
	if err := CopyDir(src, dst); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Compare
	match, reason, err := CompareDirs(src, dst)
	if err != nil {
		t.Fatalf("CompareDirs failed: %v", err)
	}
	if !match {
		t.Fatalf("CompareDirs mismatch: %s", reason)
	}

	// Modify dst and ensure mismatch detected
	if err := os.WriteFile(filepath.Join(dst, "file1.txt"), []byte("different"), 0644); err != nil {
		t.Fatal(err)
	}
	match, _, err = CompareDirs(src, dst)
	if err != nil {
		t.Fatalf("CompareDirs failed: %v", err)
	}
	if match {
		t.Fatalf("expected mismatch after modifying file")
	}

	// Test ClearDir
	if err := ClearDir(dst); err != nil {
		t.Fatalf("ClearDir failed: %v", err)
	}
	dirs, files, err := ScanDir(dst)
	if err != nil {
		t.Fatalf("ScanDir failed: %v", err)
	}
	if len(dirs) != 0 || len(files) != 0 {
		t.Fatalf("ClearDir did not clear directory, dirs: %v, files: %v", dirs, files)
	}
}
