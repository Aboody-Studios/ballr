import os
from unittest.mock import MagicMock, patch

import cv2
import numpy as np
import pytest


def _create_synthetic_video(path, num_frames=30, fps=30, width=640, height=480):
    fourcc = cv2.VideoWriter_fourcc(*"MJPG")
    writer = cv2.VideoWriter(path, fourcc, fps, (width, height))
    if not writer.isOpened():
        writer = cv2.VideoWriter(path, cv2.VideoWriter_fourcc(*"mp4v"), fps, (width, height))
    for _ in range(num_frames):
        frame = np.random.randint(0, 256, (height, width, 3), dtype=np.uint8)
        writer.write(frame)
    writer.release()
    if not os.path.isfile(path):
        pytest.skip("could not create test video file")


class TestRunPipeline:
    def test_returns_dict_with_expected_keys(self, tmp_path):
        video_path = os.path.join(tmp_path, "test.avi")
        _create_synthetic_video(video_path)

        mock_events = [{"type": "PASS", "result": "SUCCESS", "timestamp": "00:00", "coordinates": {"x": 50.0, "y": 50.0}, "insight": ""}]
        mock_summary = {"total_distance": 0.5, "top_speed": 10.0, "pass_accuracy": 1.0, "touches": 2, "sprints": 1}
        mock_heatmaps = {"overall_url": "", "defensive_url": "", "attacking_url": ""}

        mock_player = {"bbox": (100, 200, 300, 400), "confidence": 0.95, "class": "player", "center": (200, 300)}
        mock_ball = {"bbox": (400, 300, 420, 320), "confidence": 0.8, "class": "ball", "center": (410, 310)}
        patches = [
            patch("pipeline.detect_all", return_value=([mock_player], mock_ball)),
            patch("pipeline.select_target_player", return_value=mock_player),
            patch("pipeline.estimate_pose", return_value={"keypoints": [(200, 300, 0.9) for _ in range(17)], "confidence": 0.9}),
            patch("pipeline.compute_joint_angles", return_value={"left_knee": 90.0}),
            patch("pipeline.compute_head_orientation", return_value="center"),
            patch("pipeline.extract_events", return_value=mock_events),
            patch("pipeline.compute_summary", return_value=mock_summary),
            patch("pipeline.generate_all_heatmaps", return_value=mock_heatmaps),
            patch("pipeline.upload_tracking_data", return_value=""),
        ]

        for p in patches:
            p.start()

        try:
            from pipeline import run_pipeline
            result = run_pipeline(
                video_path=video_path,
                match_id="test_match",
                user_id="user1",
                shirt_number=10,
                position="FW",
            )
            assert isinstance(result, dict)
            assert "match_id" in result
            assert "summary" in result
            assert "heatmaps" in result
            assert "events" in result
            assert "tracking_data_url" in result
            assert result["match_id"] == "test_match"
            assert result["summary"] == mock_summary
            assert result["heatmaps"] == mock_heatmaps
            assert result["events"] == mock_events
        finally:
            for p in patches:
                p.stop()

    def test_raises_on_nonexistent_video(self):
        from pipeline import run_pipeline
        with pytest.raises(FileNotFoundError):
            run_pipeline(
                video_path="/nonexistent/video.avi",
                match_id="test",
                user_id="user1",
                shirt_number=10,
                position="FW",
            )

    def test_raises_on_very_short_video(self, tmp_path):
        video_path = os.path.join(tmp_path, "short.avi")
        _create_synthetic_video(video_path, num_frames=5)

        from pipeline import run_pipeline
        with pytest.raises(ValueError, match="video too short"):
            run_pipeline(
                video_path=video_path,
                match_id="test",
                user_id="user1",
                shirt_number=10,
                position="FW",
            )

    def test_returns_empty_result_when_no_player_detections(self, tmp_path):
        video_path = os.path.join(tmp_path, "test.avi")
        _create_synthetic_video(video_path, num_frames=15)

        patches = [
            patch("pipeline.detect_all", return_value=([], None)),
            patch("pipeline.select_target_player", return_value=None),
            patch("pipeline.estimate_pose", return_value=None),
            patch("pipeline.generate_all_heatmaps", return_value={"overall_url": "", "defensive_url": "", "attacking_url": ""}),
            patch("pipeline.upload_tracking_data", return_value=""),
        ]

        for p in patches:
            p.start()

        try:
            from pipeline import run_pipeline
            result = run_pipeline(
                video_path=video_path,
                match_id="test_match",
                user_id="user1",
                shirt_number=10,
                position="FW",
            )
            assert result["summary"]["total_distance"] == 0.0
            assert result["events"] == []
            assert result["heatmaps"]["overall_url"] == ""
        finally:
            for p in patches:
                p.stop()
