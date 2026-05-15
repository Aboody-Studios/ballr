package domain

import (
	"testing"
	"time"
)

func TestNewProgress(t *testing.T) {
	p := NewProgress("prog-1", "user-1")
	if p.ID != "prog-1" {
		t.Errorf("expected id prog-1, got %s", p.ID)
	}
	if p.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", p.UserID)
	}
	if p.TotalPoints != 0 {
		t.Errorf("expected 0 points, got %d", p.TotalPoints)
	}
	if p.CurrentStreak != 0 {
		t.Errorf("expected 0 streak, got %d", p.CurrentStreak)
	}
	if p.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if !p.LastActive.IsZero() {
		t.Error("expected LastActive to be zero for new progress")
	}
}

func TestAddPoints(t *testing.T) {
	t.Run("positive points", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.AddPoints(100)
		if p.TotalPoints != 100 {
			t.Errorf("expected 100 points, got %d", p.TotalPoints)
		}
	})

	t.Run("add twice", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.AddPoints(50)
		p.AddPoints(25)
		if p.TotalPoints != 75 {
			t.Errorf("expected 75 points, got %d", p.TotalPoints)
		}
	})

	t.Run("zero points ignored", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.AddPoints(0)
		if p.TotalPoints != 0 {
			t.Errorf("expected 0 points, got %d", p.TotalPoints)
		}
	})

	t.Run("negative points ignored", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.AddPoints(-50)
		if p.TotalPoints != 0 {
			t.Errorf("expected 0 points, got %d", p.TotalPoints)
		}
	})
}

func TestUpdateStreak(t *testing.T) {
	base := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	t.Run("first activity sets streak to 1", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.UpdateStreak(base)
		if p.CurrentStreak != 1 {
			t.Errorf("expected streak 1, got %d", p.CurrentStreak)
		}
		if !p.LastActive.Equal(base) {
			t.Errorf("expected LastActive %v, got %v", base, p.LastActive)
		}
	})

	t.Run("consecutive day increments streak", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.LastActive = base
		p.CurrentStreak = 1

		nextDay := base.Add(24 * time.Hour)
		p.UpdateStreak(nextDay)
		if p.CurrentStreak != 2 {
			t.Errorf("expected streak 2, got %d", p.CurrentStreak)
		}
	})

	t.Run("same day does not change streak", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.LastActive = base
		p.CurrentStreak = 5

		p.UpdateStreak(base)
		if p.CurrentStreak != 5 {
			t.Errorf("expected streak 5 unchanged, got %d", p.CurrentStreak)
		}
	})

	t.Run("gap resets streak to 1", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.LastActive = base
		p.CurrentStreak = 5

		threeDaysLater := base.Add(72 * time.Hour)
		p.UpdateStreak(threeDaysLater)
		if p.CurrentStreak != 1 {
			t.Errorf("expected streak 1 after gap, got %d", p.CurrentStreak)
		}
	})
}

func TestRecordEvent(t *testing.T) {
	t.Run("match uploaded awards 50 points", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		points := p.RecordEvent(EventMatchUploaded)
		if points != 50 {
			t.Errorf("expected 50 points, got %d", points)
		}
		if p.TotalPoints != 50 {
			t.Errorf("expected total 50, got %d", p.TotalPoints)
		}
	})

	t.Run("analysis completed awards 100 points", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		points := p.RecordEvent(EventAnalysisCompleted)
		if points != 100 {
			t.Errorf("expected 100 points, got %d", points)
		}
	})

	t.Run("coach interaction awards 10 points", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		points := p.RecordEvent(EventCoachInteraction)
		if points != 10 {
			t.Errorf("expected 10 points, got %d", points)
		}
	})

	t.Run("drill completed awards 25 points", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		points := p.RecordEvent(EventDrillCompleted)
		if points != 25 {
			t.Errorf("expected 25 points, got %d", points)
		}
	})
}

func TestNextStreakExpiry(t *testing.T) {
	base := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	p := NewProgress("p-1", "u-1")
	p.LastActive = base

	expiry := p.NextStreakExpiry()
	expected := time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC)
	if !expiry.Equal(expected) {
		t.Errorf("expected expiry %v, got %v", expected, expiry)
	}
}

func TestCanUnlockStreakAchievement(t *testing.T) {
	t.Run("7 day streak", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.CurrentStreak = 7
		typ, ok := p.CanUnlockStreakAchievement()
		if !ok {
			t.Fatal("expected unlock")
		}
		if typ != AchievementTypeStreak7 {
			t.Errorf("expected STREAK_7, got %s", typ)
		}
	})

	t.Run("30 day streak", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.CurrentStreak = 30
		typ, ok := p.CanUnlockStreakAchievement()
		if !ok {
			t.Fatal("expected unlock")
		}
		if typ != AchievementTypeStreak30 {
			t.Errorf("expected STREAK_30, got %s", typ)
		}
	})

	t.Run("no achievement at 5", func(t *testing.T) {
		p := NewProgress("p-1", "u-1")
		p.CurrentStreak = 5
		_, ok := p.CanUnlockStreakAchievement()
		if ok {
			t.Error("expected no unlock")
		}
	})
}

func TestAwardAchievement(t *testing.T) {
	p := NewProgress("p-1", "u-1")
	a := p.AwardAchievement(AchievementTypeFirstUpload)
	if a == nil {
		t.Fatal("expected non-nil achievement")
	}
	if a.Type != string(AchievementTypeFirstUpload) {
		t.Errorf("expected FIRST_UPLOAD, got %s", a.Type)
	}
	if a.PointsValue != 100 {
		t.Errorf("expected 100 points, got %d", a.PointsValue)
	}
	if a.UserID != "u-1" {
		t.Errorf("expected user u-1, got %s", a.UserID)
	}
}

func TestGetLevel(t *testing.T) {
	tests := []struct {
		points int64
		level  int
	}{
		{0, 1},
		{50, 1},
		{100, 2},
		{399, 2},
		{400, 3},
		{899, 3},
		{900, 4},
	}
	for _, tc := range tests {
		t.Run("", func(t *testing.T) {
			p := NewProgress("p-1", "u-1")
			p.TotalPoints = tc.points
			got := p.GetLevel()
			if got != tc.level {
				t.Errorf("for %d points expected level %d, got %d", tc.points, tc.level, got)
			}
		})
	}
}

func TestProgressToNextLevel(t *testing.T) {
	tests := []struct {
		points   int64
		expected int64
	}{
		{0, 100},
		{50, 50},
		{100, 300},
		{400, 500},
	}
	for _, tc := range tests {
		p := NewProgress("p-1", "u-1")
		p.TotalPoints = tc.points
		got := p.ProgressToNextLevel()
		if got != tc.expected {
			t.Errorf("for %d points expected next %d, got %d", tc.points, tc.expected, got)
		}
	}
}

func TestCalculatePoints(t *testing.T) {
	points := CalculatePoints(EventMatchUploaded, nil)
	if points != 50 {
		t.Errorf("expected 50, got %d", points)
	}
}

func TestNewEventLog(t *testing.T) {
	metadata := EventMetadata{"key": "value"}
	e := NewEventLog("user-1", EventMatchUploaded, 50, metadata)
	if e.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", e.UserID)
	}
	if string(e.Type) != string(EventMatchUploaded) {
		t.Errorf("expected MATCH_UPLOADED, got %s", e.Type)
	}
	if e.PointsAwarded != 50 {
		t.Errorf("expected 50, got %d", e.PointsAwarded)
	}
	if e.Metadata["key"] != "value" {
		t.Errorf("expected metadata value")
	}
}

func TestNewAchievement(t *testing.T) {
	a := NewAchievement("user-1", AchievementTypeStreakWeek, 150)
	if a.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", a.UserID)
	}
	if a.Type != string(AchievementTypeStreakWeek) {
		t.Errorf("expected STREAK_WEEK, got %s", a.Type)
	}
	if a.PointsValue != 150 {
		t.Errorf("expected 150, got %d", a.PointsValue)
	}
	if a.UnlockedAt.IsZero() {
		t.Error("expected UnlockedAt to be set")
	}
}

func TestAchievementPointValue(t *testing.T) {
	a := &Achievement{PointsValue: 200}
	if a.PointValue() != 200 {
		t.Errorf("expected 200, got %d", a.PointValue())
	}
}

func TestProgressError(t *testing.T) {
	err := &ProgressError{"test error"}
	if err.Error() != "test error" {
		t.Errorf("expected 'test error', got '%s'", err.Error())
	}
}

func TestTypes(t *testing.T) {
	if EventMatchUploaded != "MATCH_UPLOADED" {
		t.Errorf("unexpected event type")
	}
	if AchievementTypeFirstUpload != "FIRST_UPLOAD" {
		t.Errorf("unexpected achievement type")
	}
}

func TestPointValueMap(t *testing.T) {
	if PointValue[EventMatchUploaded] != 50 {
		t.Errorf("expected 50, got %d", PointValue[EventMatchUploaded])
	}
	if PointValue[EventAnalysisCompleted] != 100 {
		t.Errorf("expected 100, got %d", PointValue[EventAnalysisCompleted])
	}
	if PointValue[EventCoachInteraction] != 10 {
		t.Errorf("expected 10, got %d", PointValue[EventCoachInteraction])
	}
}
