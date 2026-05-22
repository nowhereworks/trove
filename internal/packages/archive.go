package packages

import "trove/internal/archives"

func generateArchive(format ArchiveFormat, files []archives.File) ([]byte, string, error) {
	switch format {
	case ArchiveTarGz:
		content, err := archives.TarGz(files)
		return content, "application/gzip", err
	case ArchiveZip:
		content, err := archives.Zip(files)
		return content, "application/zip", err
	default:
		return nil, "", ErrInvalidArchive
	}
}
