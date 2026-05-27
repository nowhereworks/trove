package updates

import (
	"context"
	"fmt"
	"strings"

	"trove/internal/packages"
)

type UpdateCheckRequest struct {
	Package        string `json:"package"`
	CurrentVersion string `json:"currentVersion"`
	CurrentDigest  string `json:"currentDigest"`
}

type UpdateCheckResponse struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	LatestVersion   string `json:"latestVersion"`
	LatestDigest    string `json:"latestDigest"`
}

type Service struct {
	store packages.Store
}

func NewService(store packages.Store) *Service {
	return &Service{store: store}
}

func (s *Service) CheckUpdate(ctx context.Context, req UpdateCheckRequest) (UpdateCheckResponse, error) {
	parts := strings.SplitN(req.Package, "/", 3)
	if len(parts) != 3 {
		return UpdateCheckResponse{}, fmt.Errorf("package must use org/namespace/package format")
	}
	org, namespace, name := parts[0], parts[1], parts[2]

	resolved, err := s.store.Resolve(ctx, org, namespace, name, "latest")
	if err != nil {
		return UpdateCheckResponse{}, err
	}

	updateAvailable := resolved.ResolvedVersion != req.CurrentVersion

	return UpdateCheckResponse{
		UpdateAvailable: updateAvailable,
		LatestVersion:   resolved.ResolvedVersion,
		LatestDigest:    resolved.Digest,
	}, nil
}
