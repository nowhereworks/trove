package cli

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	files := map[string]string{
		"dir/":         "",
		"dir/file.txt": "hello world",
		"root.txt":     "root content",
	}

	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if len(name) > 0 && name[len(name)-1] == '/' {
			header.Typeflag = tar.TypeDir
			header.Mode = 0755
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if content != "" {
			if _, err := tw.Write([]byte(content)); err != nil {
				t.Fatal(err)
			}
		}
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := extractTarGzInternal(buf.Bytes(), dest); err != nil {
		t.Fatalf("extractTarGzInternal: %v", err)
	}

	for name, content := range files {
		if len(name) > 0 && name[len(name)-1] == '/' {
			dirPath := filepath.Join(dest, name)
			if _, err := os.Stat(dirPath); err != nil {
				t.Errorf("directory %s not created: %v", name, err)
			}
		} else {
			filePath := filepath.Join(dest, name)
			data, err := os.ReadFile(filePath)
			if err != nil {
				t.Errorf("file %s not readable: %v", name, err)
				continue
			}
			if string(data) != content {
				t.Errorf("file %s content = %q, want %q", name, string(data), content)
			}
		}
	}
}

func TestExtractZip(t *testing.T) {
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)

	files := map[string]string{
		"dir/file.txt": "hello world",
		"root.txt":     "root content",
	}

	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := extractZipInternal(buf.Bytes(), dest); err != nil {
		t.Fatalf("extractZipInternal: %v", err)
	}

	for name, content := range files {
		filePath := filepath.Join(dest, name)
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("file %s not readable: %v", name, err)
			continue
		}
		if string(data) != content {
			t.Errorf("file %s content = %q, want %q", name, string(data), content)
		}
	}
}

func TestExtractArchivePathTraversal(t *testing.T) {
	buf := &bytes.Buffer{}
	gw := gzip.NewWriter(buf)
	tw := tar.NewWriter(gw)

	header := &tar.Header{
		Name:     "../../../etc/passwd",
		Mode:     0644,
		Size:     9,
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte("malicious")); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	err := extractTarGzInternal(buf.Bytes(), dest)
	if err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}
