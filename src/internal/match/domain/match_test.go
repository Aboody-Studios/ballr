package domain

import (
	"testing"
	"time"
)

func TestNewMatch(t *testing.T) {
	t.Run("valid match", func(t *testing.T) {
		m, err := NewMatch("id-1", "user-1", 10, "CM", MatchMetadata{MatchDate: time.Now()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.ID != "id-1" {
			t.Errorf("expected id id-1, got %s", m.ID)
		}
		if m.UserID != "user-1" {
			t.Errorf("expected user user-1, got %s", m.UserID)
		}
		if m.ShirtNumber != 10 {
			t.Errorf("expected shirt 10, got %d", m.ShirtNumber)
		}
		if m.Status != MatchStatusUploading {
			t.Errorf("expected status UPLOADING, got %s", m.Status)
		}
	})

	t.Run("empty ID", func(t *testing.T) {
		_, err := NewMatch("", "user-1", 10, "CM", MatchMetadata{})
		if err != ErrInvalidMatchID {
			t.Errorf("expected ErrInvalidMatchID, got %v", err)
		}
	})

	t.Run("empty userID", func(t *testing.T) {
		_, err := NewMatch("id-1", "", 10, "CM", MatchMetadata{})
		if err != ErrMissingUserID {
			t.Errorf("expected ErrMissingUserID, got %v", err)
		}
	})

	t.Run("shirt number too low", func(t *testing.T) {
		_, err := NewMatch("id-1", "user-1", 0, "CM", MatchMetadata{})
		if err != ErrInvalidShirtNumber {
			t.Errorf("expected ErrInvalidShirtNumber, got %v", err)
		}
	})

	t.Run("shirt number too high", func(t *testing.T) {
		_, err := NewMatch("id-1", "user-1", 100, "CM", MatchMetadata{})
		if err != ErrInvalidShirtNumber {
			t.Errorf("expected ErrInvalidShirtNumber, got %v", err)
		}
	})
}

func TestValidate(t *testing.T) {
	t.Run("valid match", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusUploading}
		if err := m.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid status", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: "INVALID"}
		if err := m.Validate(); err != ErrInvalidMatchStatus {
			t.Errorf("expected ErrInvalidMatchStatus, got %v", err)
		}
	})
}

func TestMarkUploadComplete(t *testing.T) {
	t.Run("from UPLOADING succeeds", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusUploading}
		if err := m.MarkUploadComplete("https://video.url"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Status != MatchStatusProcessing {
			t.Errorf("expected PROCESSING, got %s", m.Status)
		}
	})

	t.Run("from PROCESSING fails", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusProcessing}
		if err := m.MarkUploadComplete("https://video.url"); err != ErrInvalidStatusTransition {
			t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
		}
	})

	t.Run("from COMPLETED fails", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusCompleted}
		if err := m.MarkUploadComplete("https://video.url"); err != ErrInvalidStatusTransition {
			t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
		}
	})
}

func TestSetAnalysisResult(t *testing.T) {
	t.Run("from PROCESSING succeeds", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusProcessing}
		result := &AnalysisResult{Summary: AnalysisSummary{TotalDistanceKM: 10.5}}
		if err := m.SetAnalysisResult(result); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Status != MatchStatusCompleted {
			t.Errorf("expected COMPLETED, got %s", m.Status)
		}
		if m.AnalysisResult == nil {
			t.Fatal("expected non-nil analysis result")
		}
		if m.AnalysisResult.Summary.TotalDistanceKM != 10.5 {
			t.Errorf("expected 10.5 distance, got %f", m.AnalysisResult.Summary.TotalDistanceKM)
		}
	})

	t.Run("nil result fails", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusProcessing}
		if err := m.SetAnalysisResult(nil); err != ErrNilAnalysisResult {
			t.Errorf("expected ErrNilAnalysisResult, got %v", err)
		}
	})

	t.Run("from UPLOADING fails", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusUploading}
		result := &AnalysisResult{}
		if err := m.SetAnalysisResult(result); err != ErrInvalidStatusTransition {
			t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
		}
	})
}

func TestMarkFailed(t *testing.T) {
	t.Run("from PROCESSING succeeds", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusProcessing}
		if err := m.MarkFailed(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.Status != MatchStatusFailed {
			t.Errorf("expected FAILED, got %s", m.Status)
		}
	})

	t.Run("from COMPLETED fails", func(t *testing.T) {
		m := &Match{ID: "id-1", UserID: "user-1", ShirtNumber: 7, Status: MatchStatusCompleted}
		if err := m.MarkFailed(); err != ErrInvalidStatusTransition {
			t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
		}
	})
}

func TestCanViewResults(t *testing.T) {
	tests := []struct {
		name   string
		match  *Match
		expect bool
	}{
		{
			"completed with result",
			&Match{ID: "id-1", Status: MatchStatusCompleted, AnalysisResult: &AnalysisResult{}},
			true,
		},
		{
			"completed without result",
			&Match{ID: "id-1", Status: MatchStatusCompleted, AnalysisResult: nil},
			false,
		},
		{
			"processing without result",
			&Match{ID: "id-1", Status: MatchStatusProcessing, AnalysisResult: nil},
			false,
		},
		{
			"uploading without result",
			&Match{ID: "id-1", Status: MatchStatusUploading, AnalysisResult: nil},
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.match.CanViewResults()
			if got != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}

func TestGetTopInsight(t *testing.T) {
	t.Run("returns first success insight", func(t *testing.T) {
		m := &Match{
			ID:     "id-1",
			Status: MatchStatusCompleted,
			AnalysisResult: &AnalysisResult{
				Events: []MatchEvent{
					{Timestamp: "00:01", Type: EventTypePass, Result: EventResultFailure, Insight: "bad pass"},
					{Timestamp: "00:02", Type: EventTypeShot, Result: EventResultSuccess, Insight: "great shot"},
				},
			},
		}
		got := m.GetTopInsight()
		if got != "great shot" {
			t.Errorf("expected 'great shot', got '%s'", got)
		}
	})

	t.Run("no events returns empty", func(t *testing.T) {
		m := &Match{
			ID:     "id-1",
			Status: MatchStatusCompleted,
			AnalysisResult: &AnalysisResult{
				Events: []MatchEvent{},
			},
		}
		if got := m.GetTopInsight(); got != "" {
			t.Errorf("expected empty, got '%s'", got)
		}
	})

	t.Run("not completed returns empty", func(t *testing.T) {
		m := &Match{ID: "id-1", Status: MatchStatusProcessing}
		if got := m.GetTopInsight(); got != "" {
			t.Errorf("expected empty, got '%s'", got)
		}
	})
}

func TestMatchError(t *testing.T) {
	err := &MatchError{"something went wrong"}
	if err.Error() != "something went wrong" {
		t.Errorf("expected 'something went wrong', got '%s'", err.Error())
	}
}

func TestErrMatchNotFound(t *testing.T) {
	if ErrMatchNotFound.Error() != "match not found" {
		t.Errorf("expected 'match not found', got '%s'", ErrMatchNotFound.Error())
	}
}

func TestErrAnalysisNotFound(t *testing.T) {
	if ErrAnalysisNotFound.Error() != "analysis not found" {
		t.Errorf("expected 'analysis not found', got '%s'", ErrAnalysisNotFound.Error())
	}
}
