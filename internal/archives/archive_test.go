package archives

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
)

func TestTarGzIsDeterministicAndSorted(t *testing.T) {
	files := []File{{Path: "b.txt", Content: []byte("b")}, {Path: "a.txt", Content: []byte("a")}}
	one, err := TarGz(files)
	if err != nil {
		t.Fatalf("TarGz() error = %v", err)
	}
	two, err := TarGz(files)
	if err != nil {
		t.Fatalf("TarGz() error = %v", err)
	}
	if !bytes.Equal(one, two) {
		t.Fatal("TarGz() output is not deterministic")
	}

	entries := readTarGz(t, one)
	if entries[0].Path != "a.txt" || string(entries[0].Content) != "a" || entries[1].Path != "b.txt" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestZipIsDeterministicAndSorted(t *testing.T) {
	files := []File{{Path: "b.txt", Content: []byte("b")}, {Path: "a.txt", Content: []byte("a")}}
	one, err := Zip(files)
	if err != nil {
		t.Fatalf("Zip() error = %v", err)
	}
	two, err := Zip(files)
	if err != nil {
		t.Fatalf("Zip() error = %v", err)
	}
	if !bytes.Equal(one, two) {
		t.Fatal("Zip() output is not deterministic")
	}

	entries := readZip(t, one)
	if entries[0].Path != "a.txt" || string(entries[0].Content) != "a" || entries[1].Path != "b.txt" {
		t.Fatalf("entries = %+v", entries)
	}
}

func readTarGz(t *testing.T, data []byte) []File {
	t.Helper()
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("open gzip: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	var files []File
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("read tar: %v", err)
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("read tar content: %v", err)
		}
		files = append(files, File{Path: header.Name, Content: content})
	}
	return files
}

func readZip(t *testing.T, data []byte) []File {
	t.Helper()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	files := make([]File, 0, len(reader.File))
	for _, entry := range reader.File {
		file, err := entry.Open()
		if err != nil {
			t.Fatalf("open zip entry: %v", err)
		}
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read zip entry: %v", err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close zip entry: %v", err)
		}
		files = append(files, File{Path: entry.Name, Content: content})
	}
	return files
}
