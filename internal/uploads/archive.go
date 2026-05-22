package uploads

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
)

var ErrUnsafePath = errors.New("archive contains unsafe path")

type File struct {
	Path    string
	Content []byte
}

func ExtractZip(data []byte) ([]File, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}

	files := make([]File, 0, len(reader.File))
	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		name, err := safePath(entry.Name)
		if err != nil {
			return nil, err
		}

		file, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry %s: %w", entry.Name, err)
		}
		content, readErr := io.ReadAll(file)
		closeErr := file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read zip entry %s: %w", entry.Name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close zip entry %s: %w", entry.Name, closeErr)
		}
		files = append(files, File{Path: name, Content: content})
	}

	return sortFiles(files), nil
}

func ExtractTarGz(data []byte) ([]File, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("open tar.gz archive: %w", err)
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
			return nil, fmt.Errorf("read tar.gz archive: %w", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		name, err := safePath(header.Name)
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, fmt.Errorf("read tar entry %s: %w", header.Name, err)
		}
		files = append(files, File{Path: name, Content: content})
	}

	return sortFiles(files), nil
}

func safePath(name string) (string, error) {
	name = strings.TrimPrefix(strings.ReplaceAll(name, "\\", "/"), "./")
	if name == "" || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("%w: %s", ErrUnsafePath, name)
	}
	cleaned := path.Clean(name)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", fmt.Errorf("%w: %s", ErrUnsafePath, name)
	}
	return cleaned, nil
}

func sortFiles(files []File) []File {
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files
}
