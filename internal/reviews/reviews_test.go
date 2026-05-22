package reviews

import (
	"testing"
	"time"

	"trove/internal/config"
)

func TestApprovalStatusDefaults(t *testing.T) {
	svc := NewService(nil, config.ReviewsConfig{
		RequireApproval:  true,
		MinimumApprovals: 1,
	})

	status := svc.GetApprovalStatus(nil, "test-version-id")

	if status.RequiredCount != 1 {
		t.Fatalf("RequiredCount = %d, want 1", status.RequiredCount)
	}
	if status.HasEnoughApprovals {
		t.Fatal("HasEnoughApprovals should be false with no approvals")
	}
}

func TestApprovalStatusWithZeroApprovals(t *testing.T) {
	svc := NewService(nil, config.ReviewsConfig{
		RequireApproval:  true,
		MinimumApprovals: 2,
	})

	status := svc.GetApprovalStatus(nil, "test-version-id")

	if status.RequiredCount != 2 {
		t.Fatalf("RequiredCount = %d, want 2", status.RequiredCount)
	}
	if status.CurrentCount != 0 {
		t.Fatalf("CurrentCount = %d, want 0", status.CurrentCount)
	}
	if status.HasEnoughApprovals {
		t.Fatal("HasEnoughApprovals should be false with 0 approvals when 2 required")
	}
}

func TestCanPublishWhenApprovalNotRequired(t *testing.T) {
	svc := NewService(nil, config.ReviewsConfig{
		RequireApproval:  false,
		MinimumApprovals: 0,
	})

	can, err := svc.CanPublish(nil, "test-version-id")
	if err != nil {
		t.Fatalf("CanPublish error: %v", err)
	}
	if !can {
		t.Fatal("CanPublish should return true when approval not required")
	}
}

func TestCanPublishWhenApprovalRequired(t *testing.T) {
	svc := NewService(nil, config.ReviewsConfig{
		RequireApproval:  true,
		MinimumApprovals: 1,
	})

	can, err := svc.CanPublish(nil, "test-version-id")
	if err != nil {
		t.Fatalf("CanPublish error: %v", err)
	}
	if can {
		t.Fatal("CanPublish should return false when approval required but not granted")
	}
}

func TestReviewStatusConstants(t *testing.T) {
	validStatuses := []string{"approved", "changes_requested", "submitted"}

	for _, status := range validStatuses {
		switch status {
		case "approved", "changes_requested", "submitted":
			// valid
		default:
			t.Errorf("unexpected status: %s", status)
		}
	}
}

func TestFormatTime(t *testing.T) {
	// Test zero time
	got := FormatTime(time.Time{})
	if got != "" {
		t.Fatalf("FormatTime(zero) = %q, want empty", got)
	}
}
