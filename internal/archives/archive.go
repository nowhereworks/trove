package archives

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"sort"
	"time"
)

type File struct {
	Path    string
	Content []byte
}

func TarGz(files []File) ([]byte, error) {
	files = sorted(files)
	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	gzipWriter.ModTime = time.Unix(0, 0).UTC()
	tarWriter := tar.NewWriter(gzipWriter)

	for _, file := range files {
		header := &tar.Header{
			Name:    file.Path,
			Mode:    0o644,
			Size:    int64(len(file.Content)),
			ModTime: time.Unix(0, 0).UTC(),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("write tar header %s: %w", file.Path, err)
		}
		if _, err := tarWriter.Write(file.Content); err != nil {
			return nil, fmt.Errorf("write tar content %s: %w", file.Path, err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}
	return buf.Bytes(), nil
}

func Zip(files []File) ([]byte, error) {
	files = sorted(files)
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)

	for _, file := range files {
		header := &zip.FileHeader{Name: file.Path, Method: zip.Deflate}
		header.SetMode(0o644)
		header.SetModTime(time.Unix(0, 0).UTC())
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, fmt.Errorf("create zip entry %s: %w", file.Path, err)
		}
		if _, err := entry.Write(file.Content); err != nil {
			return nil, fmt.Errorf("write zip entry %s: %w", file.Path, err)
		}
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	return buf.Bytes(), nil
}

func sorted(files []File) []File {
	result := append([]File(nil), files...)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Path < result[j].Path
	})
	return result
}
