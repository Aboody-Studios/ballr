package domain

import (
	"testing"
	"time"
)

func TestNewUser(t *testing.T) {
	birthDate := time.Date(2000, 1, 15, 0, 0, 0, 0, time.UTC)
	u := NewUser("id-1", "test@example.com", "google", "https://example.com/avatar.png", "Alice", birthDate, PositionCM, FootednessRight, "improve passing")

	if u.ID != "id-1" {
		t.Errorf("expected id id-1, got %s", u.ID)
	}
	if u.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", u.Email)
	}
	if u.OAuthProvider != "google" {
		t.Errorf("expected oauth_provider google, got %s", u.OAuthProvider)
	}
	if u.AvatarURL != "https://example.com/avatar.png" {
		t.Errorf("expected avatar URL, got %s", u.AvatarURL)
	}
	if u.FullName != "Alice" {
		t.Errorf("expected name Alice, got %s", u.FullName)
	}
	if u.Position != PositionCM {
		t.Errorf("expected position CM, got %s", u.Position)
	}
	if u.Footedness != FootednessRight {
		t.Errorf("expected footedness Right, got %s", u.Footedness)
	}
	if u.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestUpdateProfile(t *testing.T) {
	u := NewUser("id-1", "test@example.com", "google", "", "Alice", time.Time{}, "", "", "")
	u.UpdateProfile("Bob", PositionST, FootednessLeft, "score more goals")

	if u.FullName != "Bob" {
		t.Errorf("expected name Bob, got %s", u.FullName)
	}
	if u.Position != PositionST {
		t.Errorf("expected position ST, got %s", u.Position)
	}
	if u.Footedness != FootednessLeft {
		t.Errorf("expected footedness Left, got %s", u.Footedness)
	}
	if u.Goals != "score more goals" {
		t.Errorf("expected goals, got %s", u.Goals)
	}
}

func TestCalculateAge(t *testing.T) {
	now := time.Now()
	thisYear := now.Year()

	tests := []struct {
		name     string
		birth    time.Time
		expected int
	}{
		{
			"birthday already passed this year",
			time.Date(thisYear-26, 1, 15, 0, 0, 0, 0, time.UTC),
			26,
		},
		{
			"birthday later this year",
			time.Date(thisYear-26, 12, 15, 0, 0, 0, 0, time.UTC),
			25,
		},
		{
			"birthday is today",
			time.Date(thisYear-26, now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
			26,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewUser("id-1", "test@example.com", "google", "", "Alice", tt.birth, "", "", "")
			got := u.CalculateAge()
			if got != tt.expected {
				t.Errorf("expected age %d (birth year %d), got %d", tt.expected, tt.birth.Year(), got)
			}
		})
	}
}

func TestNewUserZeroBirthDate(t *testing.T) {
	u := NewUser("id-1", "test@example.com", "google", "", "Alice", time.Time{}, "", "", "")
	if !u.BirthDate.IsZero() {
		t.Error("expected zero birth date")
	}
}

func TestPositionConstants(t *testing.T) {
	positions := []Position{PositionGK, PositionCB, PositionLB, PositionRB, PositionCM, PositionLW, PositionRW, PositionST}
	if len(positions) != 8 {
		t.Errorf("expected 8 positions, got %d", len(positions))
	}
}

func TestFootednessConstants(t *testing.T) {
	footedness := []Footedness{FootednessLeft, FootednessRight, FootednessBoth}
	if len(footedness) != 3 {
		t.Errorf("expected 3 footedness values, got %d", len(footedness))
	}
}
