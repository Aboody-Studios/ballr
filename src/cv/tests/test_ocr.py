"""
Tests for OCR shirt number verification module.
"""
from unittest.mock import MagicMock, patch

import numpy as np
import pytest


@pytest.fixture(autouse=True)
def reset_ocr_singleton():
    """Reset the _ocr singleton before and after each test."""
    import ocr as ocr_mod
    ocr_mod._ocr = None
    yield
    ocr_mod._ocr = None


@pytest.fixture
def sample_crop():
    """Small synthetic player crop image."""
    return np.random.randint(0, 256, (100, 50, 3), dtype=np.uint8)


def _make_ocr_result(text, confidence, bbox=None):
    """Create PaddleOCR-style result entry: [bbox, (text, confidence)]."""
    bbox = bbox or [[0, 0], [10, 0], [10, 10], [0, 10]]
    return [bbox, (text, confidence)]


class TestGetOcr:
    def test_get_ocr_import_error(self):
        """get_ocr returns None when PaddleOCR import fails."""
        import ocr as ocr_mod
        ocr_mod._ocr = None
        with patch.dict("sys.modules", {"paddleocr": MagicMock()}):
            with patch("paddleocr.PaddleOCR") as mock_paddle:
                mock_paddle.side_effect = ImportError
                result = ocr_mod.get_ocr()
                assert result is None


class TestReadShirtNumber:
    def test_extracts_numbers(self, sample_crop):
        """read_shirt_number extracts valid numeric shirt numbers."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("10", 0.95)]
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert numbers == ["10"]

    def test_extracts_multiple_numbers(self, sample_crop):
        """read_shirt_number extracts all valid numeric numbers."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [
            _make_ocr_result("10", 0.95),
            _make_ocr_result("7", 0.85),
        ]
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert numbers == ["10", "7"]

    def test_filters_non_numeric(self, sample_crop):
        """read_shirt_number filters out non-numeric text."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [
            _make_ocr_result("10", 0.95),
            _make_ocr_result("sponsor", 0.90),
            _make_ocr_result("7", 0.85),
            _make_ocr_result("adidas", 0.80),
        ]
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert "10" in numbers
        assert "7" in numbers
        assert len(numbers) == 2

    def test_filters_by_length(self, sample_crop):
        """read_shirt_number filters out text longer than 2 characters."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [
            _make_ocr_result("123", 0.95),
            _make_ocr_result("10", 0.90),
            _make_ocr_result("9999", 0.85),
        ]
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert numbers == ["10"]

    def test_empty_on_low_confidence(self, sample_crop):
        """read_shirt_number returns empty list when confidence is too low."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [
            _make_ocr_result("10", 0.3),
            _make_ocr_result("7", 0.4),
        ]
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert numbers == []

    def test_empty_on_empty_result(self, sample_crop):
        """read_shirt_number returns empty list when OCR returns no results."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = []
        numbers = ocr_mod.read_shirt_number(sample_crop, ocr_instance)
        assert numbers == []

    def test_empty_when_ocr_none(self, sample_crop):
        """read_shirt_number returns empty list when OCR is None."""
        import ocr as ocr_mod
        assert ocr_mod.read_shirt_number(sample_crop, None) == []

    def test_empty_when_crop_empty(self):
        """read_shirt_number returns empty list when crop is empty."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        empty_crop = np.array([], dtype=np.uint8).reshape(0, 0, 3)
        assert ocr_mod.read_shirt_number(empty_crop, ocr_instance) == []


class TestVerifyPlayer:
    def test_exact_match(self, sample_crop):
        """verify_player returns True when expected number matches."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("10", 0.95)]
        assert ocr_mod.verify_player(sample_crop, 10, ocr_instance) is True

    def test_no_match(self, sample_crop):
        """verify_player returns False when expected number doesn't match."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("7", 0.95)]
        assert ocr_mod.verify_player(sample_crop, 10, ocr_instance) is False

    def test_ocr_unavailable(self, sample_crop):
        """verify_player returns False when OCR is None."""
        import ocr as ocr_mod
        assert ocr_mod.verify_player(sample_crop, 10, None) is False

    def test_partial_match_last_digit(self, sample_crop):
        """verify_player matches on last digit for multi-digit expected number."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("0", 0.95)]
        assert ocr_mod.verify_player(sample_crop, 10, ocr_instance) is True

    def test_partial_match_no_false_positive_single_digit(self, sample_crop):
        """verify_player does NOT do partial match for single-digit expected number."""
        import ocr as ocr_mod
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("5", 0.95)]
        assert ocr_mod.verify_player(sample_crop, 7, ocr_instance) is False


class TestFindBestPlayerByShirt:
    def test_matches_correct_player(self, sample_crop):
        """find_best_player_by_shirt returns the matching detection."""
        import ocr as ocr_mod
        detections = [
            {"bbox": (0, 0, 50, 100), "center": (500, 300), "confidence": 0.9, "class": "player"},
            {"bbox": (100, 0, 150, 100), "center": (300, 230), "confidence": 0.85, "class": "player"},
            {"bbox": (200, 0, 250, 100), "center": (100, 100), "confidence": 0.95, "class": "player"},
        ]
        ocr_instance = MagicMock()
        ocr_instance.ocr.return_value = [_make_ocr_result("10", 0.95)]
        frame = np.zeros((480, 640, 3), dtype=np.uint8)
        with patch("ocr.get_ocr", return_value=ocr_instance), \
             patch("ocr.crop_player", return_value=sample_crop):
            result = ocr_mod.find_best_player_by_shirt(detections, frame, 10)
            assert result is not None
            assert result["center"] == (300, 230)

    def test_fallback_to_center_most(self, sample_crop):
        """find_best_player_by_shirt falls back to center-most when no match."""
        import ocr as ocr_mod
        detections = [
            {"bbox": (0, 0, 50, 100), "center": (500, 300), "confidence": 0.9, "class": "player"},
            {"bbox": (100, 0, 150, 100), "center": (300, 230), "confidence": 0.85, "class": "player"},
            {"bbox": (200, 0, 250, 100), "center": (100, 100), "confidence": 0.95, "class": "player"},
        ]
        ocr_instance = MagicMock()
        ocr_instance.ocr.side_effect = [
            [_make_ocr_result("7", 0.95)],
            [_make_ocr_result("3", 0.95)],
            [_make_ocr_result("5", 0.95)],
        ]
        frame = np.zeros((480, 640, 3), dtype=np.uint8)
        with patch("ocr.get_ocr", return_value=ocr_instance), \
             patch("ocr.crop_player", return_value=sample_crop):
            result = ocr_mod.find_best_player_by_shirt(detections, frame, 10)
            assert result is not None
            assert result["center"] == (300, 230)

    def test_no_ocr_fallback(self, sample_crop):
        """find_best_player_by_shirt returns center-most immediately when OCR unavailable."""
        import ocr as ocr_mod
        detections = [
            {"bbox": (0, 0, 50, 100), "center": (500, 300), "confidence": 0.9, "class": "player"},
            {"bbox": (100, 0, 150, 100), "center": (300, 230), "confidence": 0.85, "class": "player"},
            {"bbox": (200, 0, 250, 100), "center": (100, 100), "confidence": 0.95, "class": "player"},
        ]
        frame = np.zeros((480, 640, 3), dtype=np.uint8)
        with patch("ocr.get_ocr", return_value=None), \
             patch("ocr.crop_player", return_value=sample_crop):
            result = ocr_mod.find_best_player_by_shirt(detections, frame, 10)
            assert result is not None
            assert result["center"] == (300, 230)

    def test_returns_none_on_empty(self, sample_crop):
        """find_best_player_by_shirt returns None when detections are empty."""
        import ocr as ocr_mod
        frame = np.zeros((480, 640, 3), dtype=np.uint8)
        with patch("ocr.get_ocr", return_value=MagicMock()):
            result = ocr_mod.find_best_player_by_shirt([], frame, 10)
            assert result is None
