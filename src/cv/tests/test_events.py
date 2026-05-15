import math

import pytest

import config
from events import (
    _ball_velocity_changes,
    _min_foot_distance,
    _pitch_coords,
    compute_summary,
    detect_dribbles,
    detect_passes,
    detect_saves,
    detect_scanning,
    detect_shots,
    detect_sprints,
    detect_tackles,
    extract_events,
)


def _make_entry(
    frame_idx=0,
    player_center=(320, 240),
    ball_center=(400, 300),
    keypoints=True,
    head_orientation="center",
    player_speed=2.0,
    ball_speed=1.0,
    ball_velocity=(1.0, 0.0),
):
    kps = None
    if keypoints:
        kps = [(200.0 + i * 3, 300.0, 0.95) for i in range(17)]
    return {
        "frame_idx": frame_idx,
        "timestamp": f"00:{frame_idx // 60:02d}:{frame_idx % 60:02d}",
        "player_center": player_center,
        "ball_center": ball_center,
        "keypoints": kps,
        "head_orientation": head_orientation,
        "player_speed": player_speed,
        "player_acceleration": 0.0,
        "ball_speed": ball_speed,
        "ball_velocity": ball_velocity,
    }


def _ball_near_foot_center():
    bx = 200.0 + 15 * 3 + 15
    by = 300.0 + 5
    return (bx, by)


class TestMinFootDistance:
    def test_computes_min_distance_to_foot_keypoints(self):
        kps = [(0, 0, 0) for _ in range(17)]
        kps[13] = (100, 200, 0.95)
        kps[14] = (150, 250, 0.95)
        kps[15] = (50, 300, 0.95)
        kps[16] = (200, 350, 0.95)
        dist = _min_foot_distance((100, 200), kps)
        assert dist == pytest.approx(0.0, abs=1.0)

    def test_ignores_low_confidence_keypoints(self):
        kps = [(0, 0, 0) for _ in range(17)]
        kps[13] = (100, 200, 0.0)
        kps[14] = (100, 200, 0.0)
        dist = _min_foot_distance((100, 200), kps)
        assert dist == pytest.approx(float("inf"))


class TestBallVelocityChanges:
    def test_detects_direction_change(self):
        history = []
        for i in range(4):
            vel = (1.0, 0.0) if i < 3 else (-2.0, 0.0)
            e = _make_entry(frame_idx=i, ball_velocity=vel)
            history.append(e)
        assert _ball_velocity_changes(history, 3) is True

    def test_false_for_same_direction(self):
        history = [_make_entry(frame_idx=i, ball_velocity=(1.0, 0.0)) for i in range(4)]
        assert _ball_velocity_changes(history, 3) is False

    def test_false_when_not_enough_frames(self):
        history = [_make_entry(frame_idx=i, ball_velocity=(1.0, 0.0)) for i in range(2)]
        assert _ball_velocity_changes(history, 1) is False


class TestPitchCoords:
    def test_returns_percentage_coordinates(self):
        coords = _pitch_coords((960, 540), (1920, 1080))
        assert coords == {"x": 50.0, "y": 50.0}

    def test_edge_coordinates(self):
        coords = _pitch_coords((0, 0), (1920, 1080))
        assert coords == {"x": 0.0, "y": 0.0}


class TestDetectPasses:
    def test_detects_pass_with_velocity_change(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            vel = (3.0, 0.0) if i < 4 else (-3.0, 1.0)
            bs = 5.0 if i >= 4 else 1.0
            e = _make_entry(frame_idx=i, ball_center=ball_near, ball_speed=bs, ball_velocity=vel)
            history.append(e)
        events = detect_passes(history)
        assert len(events) == 1
        assert events[0]["type"] == "PASS"

    def test_no_pass_when_ball_not_near_foot(self):
        history = [_make_entry(frame_idx=i, ball_center=(999, 999), ball_speed=5.0) for i in range(4)]
        assert detect_passes(history) == []

    def test_no_pass_when_ball_speed_low(self):
        ball_near = _ball_near_foot_center()
        history = [_make_entry(frame_idx=i, ball_center=ball_near, ball_speed=0.5) for i in range(4)]
        assert detect_passes(history) == []


class TestDetectShots:
    def test_detects_shot_with_high_speed(self):
        ball_near = _ball_near_foot_center()
        e = _make_entry(ball_center=ball_near, ball_speed=15.0)
        events = detect_shots([_make_entry(), e])
        assert len(events) == 1
        assert events[0]["type"] == "SHOT"

    def test_no_shot_when_speed_below_threshold(self):
        ball_near = _ball_near_foot_center()
        e = _make_entry(ball_center=ball_near, ball_speed=5.0)
        assert detect_shots([_make_entry(), e]) == []

    def test_no_shot_when_ball_not_near_foot(self):
        e = _make_entry(ball_center=(999, 999), ball_speed=15.0)
        assert detect_shots([_make_entry(), e]) == []


class TestDetectDribbles:
    def test_detects_dribble_sequence(self):
        ball_near = _ball_near_foot_center()
        history = [_make_entry(frame_idx=i, ball_center=ball_near, player_speed=2.0) for i in range(5)]
        events = detect_dribbles(history)
        assert len(events) == 1
        assert events[0]["type"] == "DRIBBLE"

    def test_no_dribble_when_player_stationary(self):
        ball_near = _ball_near_foot_center()
        history = [_make_entry(frame_idx=i, ball_center=ball_near, player_speed=0.0) for i in range(5)]
        assert detect_dribbles(history) == []

    def test_no_dribble_when_ball_not_near(self):
        history = [_make_entry(frame_idx=i, ball_center=(999, 999), player_speed=2.0) for i in range(5)]
        assert detect_dribbles(history) == []


class TestDetectSprints:
    def test_detects_sprint_above_threshold(self):
        history = [_make_entry(frame_idx=i, player_speed=8.0) for i in range(15)]
        events = detect_sprints(history)
        assert len(events) == 1
        assert events[0]["type"] == "SPRINT"

    def test_no_sprint_when_below_threshold(self):
        history = [_make_entry(frame_idx=i, player_speed=3.0) for i in range(15)]
        assert detect_sprints(history) == []

    def test_no_sprint_when_not_above_threshold(self):
        history = [_make_entry(frame_idx=i, player_speed=5.0) for i in range(2)]
        assert detect_sprints(history) == []


class TestDetectTackles:
    def test_detects_tackle_with_deceleration(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            speed = 6.0 if i < 4 else 0.5
            e = _make_entry(frame_idx=i, player_speed=speed, ball_center=ball_near)
            history.append(e)
        events = detect_tackles(history)
        assert len(events) == 1
        assert events[0]["type"] == "TACKLE"

    def test_no_tackle_when_ball_too_far(self):
        history = []
        for i in range(5):
            speed = 6.0 if i < 4 else 0.5
            e = _make_entry(frame_idx=i, player_speed=speed, ball_center=(999, 999))
            history.append(e)
        assert detect_tackles(history) == []

    def test_no_tackle_without_deceleration(self):
        ball_near = _ball_near_foot_center()
        history = [_make_entry(frame_idx=i, player_speed=3.0, ball_center=ball_near) for i in range(5)]
        assert detect_tackles(history) == []


class TestDetectScanning:
    def test_detects_head_orientation_change(self):
        history = [
            _make_entry(frame_idx=0, head_orientation="center"),
            _make_entry(frame_idx=1, head_orientation="left"),
        ]
        events = detect_scanning(history)
        assert len(events) == 1
        assert events[0]["type"] == "SCANNING"

    def test_no_change_returns_empty(self):
        history = [
            _make_entry(frame_idx=0, head_orientation="center"),
            _make_entry(frame_idx=1, head_orientation="center"),
        ]
        assert detect_scanning(history) == []

    def test_unknown_orientation_skipped(self):
        history = [
            _make_entry(frame_idx=0, head_orientation="unknown"),
            _make_entry(frame_idx=1, head_orientation="left"),
        ]
        assert detect_scanning(history) == []


class TestDetectSaves:
    def test_detects_save_with_high_speed_and_deflection(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            vel = (3.0, 0.0) if i < 3 else (-2.0, 0.0)
            bs = 8.0 if i == 3 else 5.0
            bc = ball_near if i >= 3 else (999, 999)
            e = _make_entry(frame_idx=i, ball_center=bc, ball_speed=bs, ball_velocity=vel)
            history.append(e)
        events = detect_saves(history)
        assert len(events) == 1
        assert events[0]["type"] == "SAVE"

    def test_no_save_when_ball_speed_low(self):
        ball_near = _ball_near_foot_center()
        history = [_make_entry(frame_idx=i, ball_center=ball_near, ball_speed=4.0) for i in range(5)]
        assert detect_saves(history) == []

    def test_no_save_when_ball_not_near_player(self):
        history = [_make_entry(frame_idx=i, ball_center=(999, 999), ball_speed=8.0) for i in range(5)]
        assert detect_saves(history) == []

    def test_no_save_without_velocity_change(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            bs = 8.0 if i >= 3 else 5.0
            e = _make_entry(frame_idx=i, ball_center=ball_near, ball_speed=bs, ball_velocity=(3.0, 0.0))
            history.append(e)
        assert detect_saves(history) == []

    def test_save_detected_as_success_when_speed_drops(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            vel = (3.0, 0.0) if i < 3 else (-2.0, 0.0)
            bs = 15.0 if i == 2 else (7.0 if i == 3 else 5.0)
            bc = ball_near if i >= 3 else (999, 999)
            e = _make_entry(frame_idx=i, ball_center=bc, ball_speed=bs, ball_velocity=vel)
            history.append(e)
        events = detect_saves(history)
        assert len(events) == 1
        assert events[0]["result"] == "SUCCESS"

    def test_save_integration_in_extract_events(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            vel = (3.0, 0.0) if i < 3 else (-2.0, 0.0)
            bs = 8.0 if i == 3 else 5.0
            bc = ball_near if i >= 3 else (999, 999)
            ps = 2.0 if i < 4 else 0.5
            e = _make_entry(frame_idx=i, ball_center=bc, ball_speed=bs, ball_velocity=vel, player_speed=ps)
            history.append(e)
        events = extract_events(history)
        types = [ev["type"] for ev in events]
        assert "SAVE" in types


class TestExtractEvents:
    def test_returns_sorted_events(self):
        ball_near = _ball_near_foot_center()
        history = []
        for i in range(5):
            vel = (2.0, 0.0) if i < 3 else (1.0, 0.0)
            bs = 13.0 if i >= 3 else 1.0
            e = _make_entry(frame_idx=i, ball_center=ball_near, ball_speed=bs, ball_velocity=vel, player_speed=8.0)
            history.append(e)
        events = extract_events(history)
        assert isinstance(events, list)

    def test_no_events_with_empty_history(self):
        assert extract_events([]) == []

    def test_no_events_with_no_ball_stationary(self):
        history = [_make_entry(frame_idx=i, ball_center=None, keypoints=False, player_speed=0.0, head_orientation="unknown") for i in range(3)]
        events = extract_events(history)
        assert len(events) == 0


class TestComputeSummary:
    def test_returns_correct_structure(self, tracking_history):
        events = [{"type": "PASS", "result": "SUCCESS", "timestamp": "00:00", "coordinates": {"x": 50, "y": 50}, "insight": ""}]
        summary = compute_summary(tracking_history, events)
        assert "total_distance" in summary
        assert "top_speed" in summary
        assert "pass_accuracy" in summary
        assert "touches" in summary
        assert "sprints" in summary

    def test_pass_accuracy_with_mixed_results(self):
        history = [_make_entry(frame_idx=i) for i in range(3)]
        events = [
            {"type": "PASS", "result": "SUCCESS", "timestamp": "00:00"},
            {"type": "PASS", "result": "FAILURE", "timestamp": "00:01"},
        ]
        summary = compute_summary(history, events)
        assert summary["pass_accuracy"] == 0.5

    def test_pass_accuracy_no_passes(self):
        summary = compute_summary([_make_entry()], [])
        assert summary["pass_accuracy"] == 0.0

    def test_empty_history_returns_zeros(self):
        summary = compute_summary([], [])
        assert summary["total_distance"] == 0.0
        assert summary["top_speed"] == 0.0
        assert summary["touches"] == 0
        assert summary["sprints"] == 0

    def test_save_event_counts_as_touch(self):
        history = [_make_entry(frame_idx=i) for i in range(3)]
        events = [
            {"type": "SAVE", "result": "SUCCESS", "timestamp": "00:00"},
        ]
        summary = compute_summary(history, events)
        assert summary["touches"] == 1
