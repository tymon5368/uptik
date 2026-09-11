package updater

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.1", "1.0.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.2.0", "1.1.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"0.9.5", "1.0.0", -1},
		{"v1.0.0-beta.1", "1.0.0", 0}, // Base semver match
		{"1.10.0", "1.9.0", 1},
	}

	for _, tt := range tests {
		got := CompareSemver(tt.v1, tt.v2)
		if got != tt.expected {
			t.Errorf("CompareSemver(%q, %q) = %d; want %d", tt.v1, tt.v2, got, tt.expected)
		}
	}
}

func TestExtractBinaryFromTarGz(t *testing.T) {
	// Create synthetic tar.gz with "uptik" executable
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	content := []byte("#!/bin/sh\necho updated\n")
	hdr := &tar.Header{
		Name: "uptik-1.0.1-linux-amd64/uptik",
		Mode: 0755,
		Size: int64(len(content)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	tw.Close()
	gzw.Close()

	tmpArchive, err := os.CreateTemp("", "test-archive-*.tar.gz")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpArchive.Name())
	if _, err := tmpArchive.Write(buf.Bytes()); err != nil {
		t.Fatalf("Write temp failed: %v", err)
	}
	tmpArchive.Close()

	binPath, err := extractBinary(tmpArchive.Name(), "uptik-1.0.1-linux-amd64.tar.gz")
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}
	defer os.Remove(binPath)

	extracted, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(extracted, content) {
		t.Fatalf("Extracted content mismatch: got %s, want %s", string(extracted), string(content))
	}
}

func TestExtractBinaryFromZip(t *testing.T) {
	// Create synthetic zip with "uptik.exe"
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	content := []byte("MZ-executable-binary-mock")
	w, err := zw.Create("uptik-1.0.1-windows-amd64/uptik.exe")
	if err != nil {
		t.Fatalf("zip.Create failed: %v", err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatalf("zip.Write failed: %v", err)
	}
	zw.Close()

	tmpArchive, err := os.CreateTemp("", "test-archive-*.zip")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer os.Remove(tmpArchive.Name())
	if _, err := tmpArchive.Write(buf.Bytes()); err != nil {
		t.Fatalf("Write temp failed: %v", err)
	}
	tmpArchive.Close()

	binPath, err := extractBinary(tmpArchive.Name(), "uptik-1.0.1-windows-amd64-portable.zip")
	if err != nil {
		t.Fatalf("extractBinary failed: %v", err)
	}
	defer os.Remove(binPath)

	extracted, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if !bytes.Equal(extracted, content) {
		t.Fatalf("Extracted content mismatch: got %s, want %s", string(extracted), string(content))
	}
}

func TestReplaceExecutable(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "uptik")
	newPath := filepath.Join(dir, "uptik_new")

	if err := os.WriteFile(targetPath, []byte("old-binary"), 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile(newPath, []byte("new-binary"), 0755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := replaceExecutable(targetPath, newPath); err != nil {
		t.Fatalf("replaceExecutable failed: %v", err)
	}

	finalContent, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(finalContent) != "new-binary" {
		t.Fatalf("Target not replaced: got %s, want new-binary", string(finalContent))
	}
}
