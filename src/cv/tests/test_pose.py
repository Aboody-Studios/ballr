from unittest.mock import MagicMock, patch

import numpy as np
import pytest

from pose import (
    COCO_KEYPOINT_NAMES,
    _compute_iou,
    compute_angle,
    compute_head_orientation,
    compute_joint_angles,
    estimate_pose,
)


def _make_keypoint_result(kps_3d):
    mock = MagicMock()
    arr = np.array(kps_3d, dtype=float)
    cpu_mock = MagicMock()
    cpu_mock.numpy.return_value = arr
    mock.keypoints = MagicMock()
    mock.keypoints.data = MagicMock()
    mock.keypoints.data.cpu.return_value = cpu_mock
    return mock


class TestComputeIOU:
    def test_overlapping_boxes(self):
        iou = _compute_iou((0, 0, 10, 10), (5, 5, 15, 15))
        assert iou > 0
        assert iou < 1

    def test_non_overlapping_returns_zero(self):
        iou = _compute_iou((0, 0, 10, 10), (20, 20, 30, 30))
        assert iou == 0.0

    def test_identical_boxes(self):
        iou = _compute_iou((0, 0, 10, 10), (0, 0, 10, 10))
        assert iou == 1.0


class TestComputeAngle:
    def test_right_angle(self):
        a, b, c = (0, 0), (0, 1), (1, 1)
        angle = compute_angle(a, b, c)
        assert angle == pytest.approx(90.0, abs=1.0)

    def test_straight_line_collinear(self):
        a, b, c = (0, 0), (0, 1), (0, 2)
        angle = compute_angle(a, b, c)
        assert angle == pytest.approx(180.0, abs=0.1)

    def test_zero_length_vectors(self):
        angle = compute_angle((0, 0), (0, 0), (1, 1))
        assert angle == 0.0


class TestEstimatePose:
    def test_returns_keypoints(self, synthetic_frame, mock_pose_model):
        with patch("pose.get_pose_model", return_value=mock_pose_model):
            result = estimate_pose(synthetic_frame, (0, 0, 100, 100))
            assert result is not None
            assert "keypoints" in result
            assert "confidence" in result
            assert len(result["keypoints"]) == 17

    def test_returns_none_on_no_results(self, synthetic_frame):
        model = MagicMock()
        model.return_value = []
        with patch("pose.get_pose_model", return_value=model):
            assert estimate_pose(synthetic_frame, (0, 0, 100, 100)) is None

    def test_returns_none_when_no_keypoints(self, synthetic_frame):
        model = MagicMock()
        result = MagicMock()
        result.keypoints = None
        model.return_value = [result]
        with patch("pose.get_pose_model", return_value=model):
            assert estimate_pose(synthetic_frame, (0, 0, 100, 100)) is None

    def test_picks_best_person_by_iou(self, synthetic_frame):
        kps_1 = [[100 + j * 3, 200 + j * 2, 0.9] for j in range(17)]
        kps_2 = [[300 + j * 3, 400 + j * 2, 0.9] for j in range(17)]
        kps_data = [kps_1, kps_2]
        result = _make_keypoint_result(kps_data)
        model = MagicMock()
        model.return_value = [result]
        with patch("pose.get_pose_model", return_value=model):
            pose = estimate_pose(synthetic_frame, (100, 200, 150, 230))
            assert pose is not None
            xs = [kp[0] for kp in pose["keypoints"] if kp[2] > 0]
            assert min(xs) >= 90


class TestComputeJointAngles:
    def test_returns_dict_with_known_angle_names(self, mock_pose_keypoints):
        angles = compute_joint_angles(mock_pose_keypoints)
        assert isinstance(angles, dict)
        expected_names = [
            "left_knee", "right_knee", "left_hip", "right_hip",
            "left_shoulder", "right_shoulder", "left_elbow", "right_elbow",
            "left_ankle", "right_ankle",
        ]
        for name in expected_names:
            assert name in angles

    def test_angles_are_positive_floats(self, mock_pose_keypoints):
        angles = compute_joint_angles(mock_pose_keypoints)
        for name, val in angles.items():
            assert isinstance(val, float)
            assert 0.0 <= val <= 180.0

    def test_empty_when_all_keypoints_have_zero_conf(self):
        kps = [(0, 0, 0) for _ in range(17)]
        assert compute_joint_angles(kps) == {}


class TestComputeHeadOrientation:
    def test_nose_left_of_shoulder_mid(self):
        kps = [(100, 200, 0.9) for _ in range(17)]
        kps[5] = (200, 250, 0.9)
        kps[6] = (300, 250, 0.9)
        assert compute_head_orientation(kps) == "left"

    def test_nose_right_of_shoulder_mid(self):
        kps = [(300, 200, 0.9) for _ in range(17)]
        kps[5] = (200, 250, 0.9)
        kps[6] = (250, 250, 0.9)
        assert compute_head_orientation(kps) == "right"

    def test_nose_center_of_shoulders(self):
        kps = [(200, 200, 0.9) for _ in range(17)]
        kps[5] = (150, 250, 0.9)
        kps[6] = (250, 250, 0.9)
        assert compute_head_orientation(kps) == "center"

    def test_unknown_when_low_confidence(self):
        kps = [(200, 200, 0) for _ in range(17)]
        assert compute_head_orientation(kps) == "unknown"

    def test_unknown_when_fewer_than_17_keypoints(self):
        assert compute_head_orientation([(0, 0, 0)]) == "unknown"
