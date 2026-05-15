from unittest.mock import MagicMock, patch

import numpy as np
import pytest

from detector import (
    _parse_yolo_boxes,
    crop_player,
    detect_ball,
    detect_players,
    select_target_player,
)


class _IndexableMock:
    def __init__(self, items):
        self._items = items

    def __getitem__(self, i):
        m = MagicMock()
        item = self._items[i]
        if isinstance(item, list):
            m.tolist.return_value = item
        else:
            m.__float__ = lambda _=float(item): float(item)
            m.__int__ = lambda _=int(item): int(item)
            m.tolist.return_value = [item]
        return m

    def __len__(self):
        return len(self._items)

    def __bool__(self):
        return len(self._items) > 0


class _BoxesMock:
    def __init__(self, xyxy, conf, cls, names=None):
        self.xyxy = _IndexableMock(xyxy)
        self.conf = _IndexableMock(conf)
        self.cls = _IndexableMock(cls)
        self.names = names or {}
        self._box_len = len(xyxy)

    def __len__(self):
        return self._box_len


def _make_result(boxes, names):
    r = MagicMock()
    r.boxes = boxes
    r.names = names
    return r


class TestParseYoloBoxes:
    def test_returns_list_of_detections(self):
        boxes = _BoxesMock(
            [[10.0, 20.0, 100.0, 200.0], [50.0, 60.0, 150.0, 250.0]],
            [0.9, 0.8],
            [0, 1],
        )
        result = _make_result(boxes, {0: "player", 1: "ball"})
        dets = _parse_yolo_boxes(result)
        assert len(dets) == 2
        assert dets[0]["class"] == "player"
        assert dets[1]["class"] == "ball"
        assert dets[0]["center"] == (55.0, 110.0)
        assert dets[1]["center"] == (100.0, 155.0)

    def test_empty_boxes_returns_empty(self):
        boxes = _BoxesMock([], [], [])
        result = _make_result(boxes, {})
        assert _parse_yolo_boxes(result) == []

    def test_no_boxes_attr_returns_empty(self):
        result = MagicMock()
        result.boxes = None
        assert _parse_yolo_boxes(result) == []


class TestDetectPlayers:
    def test_returns_player_detections(self, synthetic_frame, mock_model):
        with patch("detector.get_detection_model", return_value=mock_model):
            players = detect_players(synthetic_frame, conf_threshold=0.5)
            assert isinstance(players, list)
            assert len(players) == 3
            for p in players:
                assert p["class"] in ("player", "goalkeeper")
                assert "bbox" in p
                assert "confidence" in p
                assert "center" in p

    def test_filters_by_confidence(self, synthetic_frame):
        model = MagicMock()
        boxes = _BoxesMock([[10.0, 20.0, 100.0, 200.0]], [0.3], [0])
        result = _make_result(boxes, {0: "player"})
        model.return_value = [result]
        with patch("detector.get_detection_model", return_value=model):
            players = detect_players(synthetic_frame, conf_threshold=0.5)
            assert len(players) == 0

    def test_empty_results_returns_empty(self, synthetic_frame):
        model = MagicMock()
        model.return_value = []
        with patch("detector.get_detection_model", return_value=model):
            assert detect_players(synthetic_frame) == []


class TestDetectBall:
    def test_returns_ball_detection(self, synthetic_frame):
        model = MagicMock()
        boxes = _BoxesMock([[400.0, 300.0, 420.0, 320.0]], [0.8], [1])
        result = _make_result(boxes, {0: "player", 1: "ball"})
        model.return_value = [result]
        with patch("detector.get_detection_model", return_value=model), \
             patch("detector.get_ball_model", return_value=model):
            ball = detect_ball(synthetic_frame, conf_threshold=0.5)
            assert ball is not None
            assert ball["class"] == "ball"

    def test_returns_none_when_no_ball(self, synthetic_frame, mock_model):
        with patch("detector.get_detection_model", return_value=mock_model), \
             patch("detector.get_ball_model", return_value=mock_model):
            ball = detect_ball(synthetic_frame, conf_threshold=0.99)
            assert ball is None

    def test_returns_none_on_empty_result(self, synthetic_frame):
        model = MagicMock()
        model.return_value = []
        with patch("detector.get_detection_model", return_value=model), \
             patch("detector.get_ball_model", return_value=model):
            assert detect_ball(synthetic_frame) is None


class TestSelectTargetPlayer:
    def test_picks_center_most_when_multiple(self):
        detections = [
            {"center": (100, 100), "confidence": 0.9, "class": "player", "bbox": (0, 0, 200, 200)},
            {"center": (320, 240), "confidence": 0.8, "class": "player", "bbox": (300, 200, 340, 280)},
            {"center": (600, 400), "confidence": 0.95, "class": "player", "bbox": (500, 300, 700, 500)},
        ]
        target = select_target_player(detections, frame_center=(320, 240))
        assert target is not None
        assert target["center"] == (320, 240)

    def test_returns_none_on_empty(self):
        assert select_target_player([]) is None

    def test_returns_single_detection(self):
        det = {"center": (100, 100), "confidence": 0.9, "class": "player", "bbox": (0, 0, 200, 200)}
        assert select_target_player([det]) is det

    def test_prefers_higher_confidence_when_equidistant(self):
        detections = [
            {"center": (300, 200), "confidence": 0.7, "class": "player", "bbox": (200, 100, 400, 300)},
            {"center": (340, 280), "confidence": 0.95, "class": "player", "bbox": (300, 200, 380, 360)},
        ]
        target = select_target_player(detections, frame_center=(320, 240))
        assert target["confidence"] == 0.95


class TestCropPlayer:
    def test_returns_valid_crop(self, synthetic_frame):
        bbox = (100, 100, 300, 300)
        crop = crop_player(synthetic_frame, bbox)
        assert crop.shape == (200, 200, 3)

    def test_clamps_to_frame_boundaries(self, synthetic_frame):
        bbox = (-50, -50, 1000, 1000)
        crop = crop_player(synthetic_frame, bbox)
        assert crop.shape[0] <= synthetic_frame.shape[0]
        assert crop.shape[1] <= synthetic_frame.shape[1]

    def test_invalid_bbox_fallback(self, synthetic_frame):
        bbox = (0, 0, 0, 0)
        crop = crop_player(synthetic_frame, bbox)
        assert crop is not None
        assert crop.ndim == 3
