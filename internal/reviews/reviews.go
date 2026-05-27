package reviews

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"trove/internal/config"
	"trove/internal/db/sqlc"
	"trove/internal/packages"
)

var (
	ErrReviewNotFound        = errors.New("review not found")
	ErrAlreadyApproved       = errors.New("already approved by this reviewer")
	ErrSelfApproval          = errors.New("self-approval is not allowed")
	ErrInsufficientApprovals = errors.New("insufficient approvals")
	ErrInvalidReviewer       = errors.New("reviewer must be a user")
	ErrVersionNotSubmittable = errors.New("version is not submittable")
)

type Service struct {
	queries *sqlc.Queries
	cfg     config.ReviewsConfig
}

func NewService(queries *sqlc.Queries, cfg config.ReviewsConfig) *Service {
	return &Service{
		queries: queries,
		cfg:     cfg,
	}
}

type Review struct {
	ID               string `json:"id"`
	PackageVersionID string `json:"packageVersionId"`
	ReviewerID       string `json:"reviewerId"`
	Status           string `json:"status"`
	Comment          string `json:"comment,omitempty"`
	CreatedAt        string `json:"createdAt"`
	UpdatedAt        string `json:"updatedAt"`
}

type ApprovalStatus struct {
	HasEnoughApprovals bool  `json:"hasEnoughApprovals"`
	CurrentCount       int64 `json:"currentCount"`
	RequiredCount      int   `json:"requiredCount"`
}

type QueueItem struct {
	Org                string `json:"org"`
	Namespace          string `json:"namespace"`
	Package            string `json:"package"`
	DisplayName        string `json:"displayName"`
	Description        string `json:"description"`
	Visibility         string `json:"visibility"`
	Version            string `json:"version"`
	Lifecycle          string `json:"lifecycle"`
	Digest             string `json:"digest"`
	ReviewID           string `json:"reviewId"`
	PackageVersionID   string `json:"packageVersionId"`
	ReviewerID         string `json:"reviewerId"`
	ReviewStatus       string `json:"reviewStatus"`
	CurrentApprovals   int64  `json:"currentApprovals"`
	RequiredApprovals  int    `json:"requiredApprovals"`
	HasEnoughApprovals bool   `json:"hasEnoughApprovals"`
	CreatedAt          string `json:"createdAt"`
	UpdatedAt          string `json:"updatedAt"`
}

func (s *Service) ResolvePackageVersionID(ctx context.Context, org, namespace, packageName, version string) (string, error) {
	row, err := s.queries.GetPackageVersionIDForProjectInstall(ctx, sqlc.GetPackageVersionIDForProjectInstallParams{
		Org:         org,
		Namespace:   namespace,
		PackageName: packageName,
		Version:     version,
	})
	if err != nil {
		return "", err
	}
	return uuid.UUID(row.Bytes).String(), nil
}

func (s *Service) PackageVersionIDForReview(ctx context.Context, reviewID string) (string, error) {
	review, err := s.queries.GetReview(ctx, mustParseUUID(reviewID))
	if err != nil {
		return "", err
	}
	return uuid.UUID(review.PackageVersionID.Bytes).String(), nil
}

func (s *Service) SubmitForReview(ctx context.Context, packageVersionID, userID string) error {
	id, _ := uuid.NewV7()
	pvID, err := uuid.Parse(packageVersionID)
	if err != nil {
		return packages.ErrVersionNotFound
	}
	rID, err := uuid.Parse(userID)
	if err != nil || userID == "" {
		return ErrInvalidReviewer
	}

	_, err = s.queries.CreateReview(ctx, sqlc.CreateReviewParams{
		ID:               pgtype.UUID{Bytes: id, Valid: true},
		PackageVersionID: pgtype.UUID{Bytes: pvID, Valid: true},
		ReviewerID:       pgtype.UUID{Bytes: rID, Valid: true},
		Status:           "submitted",
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrVersionNotSubmittable
	}
	return err
}

func (s *Service) Approve(ctx context.Context, reviewID, reviewerID, packageVersionID, comment string) (Review, error) {
	existing, err := s.queries.GetReview(ctx, mustParseUUID(reviewID))
	if err != nil {
		return Review{}, err
	}

	reviewerUUID := mustParseUUID(reviewerID)
	if !s.cfg.AllowSelfApproval && existing.ReviewerID == reviewerUUID {
		return Review{}, ErrSelfApproval
	}
	hasApproval, err := s.queries.HasApprovalFrom(ctx, sqlc.HasApprovalFromParams{
		PackageVersionID: mustParseUUID(packageVersionID),
		ReviewerID:       reviewerUUID,
	})
	if err == nil && hasApproval {
		return Review{}, ErrAlreadyApproved
	}

	rID := mustParseUUID(reviewID)

	pgComment := pgtype.Text{}
	if comment != "" {
		pgComment = pgtype.Text{String: comment, Valid: true}
	}

	row, err := s.queries.UpdateReviewStatus(ctx, sqlc.UpdateReviewStatusParams{
		ID:      rID,
		Status:  "approved",
		Comment: pgComment,
	})
	if err != nil {
		return Review{}, err
	}

	approvalID, _ := uuid.NewV7()
	err = s.queries.UpsertApproval(ctx, sqlc.UpsertApprovalParams{
		ID:               pgtype.UUID{Bytes: approvalID, Valid: true},
		PackageVersionID: mustParseUUID(packageVersionID),
		ReviewerID:       reviewerUUID,
	})
	if err != nil {
		return Review{}, err
	}

	return toReview(row), nil
}

func (s *Service) ListQueue(ctx context.Context, limit int) ([]QueueItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := s.queries.ListReviewQueue(ctx, int32(limit))
	if err != nil {
		return nil, err
	}

	items := make([]QueueItem, 0, len(rows))
	for _, row := range rows {
		required := s.cfg.MinimumApprovals
		items = append(items, QueueItem{
			Org:                row.Org,
			Namespace:          row.Namespace,
			Package:            row.PackageName,
			DisplayName:        row.DisplayName,
			Description:        row.Description,
			Visibility:         row.Visibility,
			Version:            row.Version,
			Lifecycle:          row.Lifecycle,
			Digest:             row.Digest,
			ReviewID:           uuid.UUID(row.ReviewID.Bytes).String(),
			PackageVersionID:   uuid.UUID(row.PackageVersionID.Bytes).String(),
			ReviewerID:         uuid.UUID(row.ReviewerID.Bytes).String(),
			ReviewStatus:       row.ReviewStatus,
			CurrentApprovals:   row.CurrentApprovals,
			RequiredApprovals:  required,
			HasEnoughApprovals: row.CurrentApprovals >= int64(required),
			CreatedAt:          packages.FormatTime(row.CreatedAt.Time),
			UpdatedAt:          packages.FormatTime(row.UpdatedAt.Time),
		})
	}
	return items, nil
}

func (s *Service) RequestChanges(ctx context.Context, reviewID, comment string) (Review, error) {
	rID := mustParseUUID(reviewID)

	pgComment := pgtype.Text{}
	if comment != "" {
		pgComment = pgtype.Text{String: comment, Valid: true}
	}

	row, err := s.queries.UpdateReviewStatus(ctx, sqlc.UpdateReviewStatusParams{
		ID:      rID,
		Status:  "changes_requested",
		Comment: pgComment,
	})
	if err != nil {
		return Review{}, err
	}

	return toReview(row), nil
}

func (s *Service) AddComment(ctx context.Context, reviewID, authorID, body string) (ReviewComment, error) {
	id, _ := uuid.NewV7()
	row, err := s.queries.AddReviewComment(ctx, sqlc.AddReviewCommentParams{
		ID:       pgtype.UUID{Bytes: id, Valid: true},
		ReviewID: mustParseUUID(reviewID),
		AuthorID: mustParseUUID(authorID),
		Body:     body,
	})
	if err != nil {
		return ReviewComment{}, err
	}

	return ReviewComment{
		ID:        uuid.UUID(row.ID.Bytes).String(),
		ReviewID:  uuid.UUID(row.ReviewID.Bytes).String(),
		AuthorID:  uuid.UUID(row.AuthorID.Bytes).String(),
		Body:      row.Body,
		CreatedAt: packages.FormatTime(row.CreatedAt.Time),
	}, nil
}

func (s *Service) ListReviews(ctx context.Context, packageVersionID string) ([]Review, error) {
	rows, err := s.queries.ListReviewsForVersion(ctx, mustParseUUID(packageVersionID))
	if err != nil {
		return nil, err
	}

	reviews := make([]Review, len(rows))
	for i, row := range rows {
		reviews[i] = toReview(row)
	}
	return reviews, nil
}

func (s *Service) GetApprovalStatus(ctx context.Context, packageVersionID string) ApprovalStatus {
	if s.queries == nil {
		return ApprovalStatus{RequiredCount: s.cfg.MinimumApprovals}
	}
	count, err := s.queries.CountApprovalsForVersion(ctx, mustParseUUID(packageVersionID))
	if err != nil {
		return ApprovalStatus{RequiredCount: s.cfg.MinimumApprovals}
	}

	return ApprovalStatus{
		HasEnoughApprovals: count >= int64(s.cfg.MinimumApprovals),
		CurrentCount:       count,
		RequiredCount:      s.cfg.MinimumApprovals,
	}
}

func (s *Service) CanPublish(ctx context.Context, packageVersionID string) (bool, error) {
	if !s.cfg.RequireApproval {
		return true, nil
	}

	status := s.GetApprovalStatus(ctx, packageVersionID)
	return status.HasEnoughApprovals, nil
}

func (s *Service) InvalidateApprovals(ctx context.Context, packageVersionID string) error {
	rows, err := s.queries.ListReviewsForVersion(ctx, mustParseUUID(packageVersionID))
	if err != nil {
		return err
	}

	for _, row := range rows {
		if row.Status == "approved" {
			_ = s.queries.RemoveApproval(ctx, sqlc.RemoveApprovalParams{
				PackageVersionID: row.PackageVersionID,
				ReviewerID:       row.ReviewerID,
			})
		}
	}

	return nil
}

type ReviewComment struct {
	ID        string `json:"id"`
	ReviewID  string `json:"reviewId"`
	AuthorID  string `json:"authorId"`
	Body      string `json:"body"`
	CreatedAt string `json:"createdAt"`
}

func toReview(row sqlc.Review) Review {
	comment := ""
	if row.Comment.Valid {
		comment = row.Comment.String
	}

	return Review{
		ID:               uuid.UUID(row.ID.Bytes).String(),
		PackageVersionID: uuid.UUID(row.PackageVersionID.Bytes).String(),
		ReviewerID:       uuid.UUID(row.ReviewerID.Bytes).String(),
		Status:           row.Status,
		Comment:          comment,
		CreatedAt:        packages.FormatTime(row.CreatedAt.Time),
		UpdatedAt:        packages.FormatTime(row.UpdatedAt.Time),
	}
}

func mustParseUUID(s string) pgtype.UUID {
	uid, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}
	}
	return pgtype.UUID{Bytes: uid, Valid: true}
}

func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
