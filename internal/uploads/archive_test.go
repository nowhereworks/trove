package uploads

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"testing"
)

func TestExtractZip(t *testing.T) {
	data := makeZip(t, map[string]string{"b.txt": "b", "a.txt": "a", "dir/": ""})
	files, err := ExtractZip(data)
	if err != nil {
		t.Fatalf("ExtractZip() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Path != "a.txt" || string(files[0].Content) != "a" || files[1].Path != "b.txt" {
		t.Fatalf("files = %+v", files)
	}
}

func TestExtractTarGz(t *testing.T) {
	data := makeTarGz(t, map[string]string{"nested/b.txt": "b", "a.txt": "a"})
	files, err := ExtractTarGz(data)
	if err != nil {
		t.Fatalf("ExtractTarGz() error = %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(files))
	}
	if files[0].Path != "a.txt" || files[1].Path != "nested/b.txt" || string(files[1].Content) != "b" {
		t.Fatalf("files = %+v", files)
	}
}

func TestExtractRejectsUnsafePaths(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		fn   func([]byte) ([]File, error)
	}{
		{name: "zip parent", data: makeZip(t, map[string]string{"../escape.txt": "x"}), fn: ExtractZip},
		{name: "zip absolute", data: makeZip(t, map[string]string{"/escape.txt": "x"}), fn: ExtractZip},
		{name: "tar parent", data: makeTarGz(t, map[string]string{"../escape.txt": "x"}), fn: ExtractTarGz},
		{name: "tar absolute", data: makeTarGz(t, map[string]string{"/escape.txt": "x"}), fn: ExtractTarGz},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fn(tt.data)
			if !errors.Is(err, ErrUnsafePath) {
				t.Fatalf("error = %v, want ErrUnsafePath", err)
			}
		})
	}
}

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		if name[len(name)-1] == '/' {
			if _, err := writer.Create(name); err != nil {
				t.Fatalf("create zip dir: %v", err)
			}
			continue
		}
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip writer: %v", err)
	}
	return buf.Bytes()
}

func makeTarGz(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		data := []byte(content)
		if err := tarWriter.WriteHeader(&tar.Header{Name: name, Mode: 0o600, Size: int64(len(data))}); err != nil {
			t.Fatalf("write tar header: %v", err)
		}
		if _, err := tarWriter.Write(data); err != nil {
			t.Fatalf("write tar content: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("close tar writer: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("close gzip writer: %v", err)
	}
	return buf.Bytes()
}
