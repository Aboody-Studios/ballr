import gc as gc_module
from unittest.mock import MagicMock, patch

import numpy as np
import pytest


def _make_corner_keypoints(fw=1920, fh=1080):
    kps = np.zeros((1, 32, 3), dtype=np.float32)
    kps[0, 0] = [0.0, 0.0, 0.95]
    kps[0, 1] = [fw - 1, 0.0, 0.95]
    kps[0, 2] = [fw - 1, fh - 1, 0.95]
    kps[0, 3] = [0.0, fh - 1, 0.95]
    for i in range(4, 32):
        kps[0, i] = [fw / 2, fh / 2, 0.1]
    return kps


def _make_mock_pitch_result(keypoints_array):
    cpu_mock = MagicMock()
    cpu_mock.numpy.return_value = keypoints_array

    mock_kp = MagicMock()
    mock_kp.data = MagicMock()
    mock_kp.data.cpu.return_value = cpu_mock

    result = MagicMock()
    result.keypoints = mock_kp
    return result


def _make_field_keypoints():
    kps = np.zeros((1, 32, 3), dtype=np.float32)
    kps[0, 0] = [200.0, 100.0, 0.95]
    kps[0, 1] = [1700.0, 100.0, 0.95]
    kps[0, 2] = [1700.0, 900.0, 0.95]
    kps[0, 3] = [200.0, 900.0, 0.95]
    for i in range(4, 32):
        kps[0, i] = [950.0, 500.0, 0.95]
    return kps


class TestDetectPitchKeypoints:
    def test_returns_list_of_keypoints(self, synthetic_frame):
        kp_data = np.zeros((1, 32, 3), dtype=np.float32)
        for i in range(32):
            kp_data[0, i] = [100.0 + i * 10, 200.0 + i * 5, 0.95 - i * 0.02]

        mock_cpu = MagicMock()
        mock_cpu.numpy.return_value = kp_data
        mock_kp = MagicMock()
        mock_kp.data = MagicMock()
        mock_kp.data.cpu.return_value = mock_cpu
        mock_result = MagicMock()
        mock_result.keypoints = mock_kp
        mock_model = MagicMock()
        mock_model.return_value = [mock_result]

        import pitch as pitch_module

        with patch.object(pitch_module, "get_pitch_model", return_value=mock_model):
            kps = pitch_module.detect_pitch_keypoints(synthetic_frame)
            assert len(kps) == 32
            assert all(len(kp) == 3 for kp in kps)
            assert kps[0][0] == pytest.approx(100.0)
            assert kps[0][2] == pytest.approx(0.95)

    def test_returns_empty_when_no_keypoints(self, synthetic_frame):
        mock_result = MagicMock()
        mock_result.keypoints = None
        mock_model = MagicMock()
        mock_model.return_value = [mock_result]

        import pitch as pitch_module

        with patch.object(pitch_module, "get_pitch_model", return_value=mock_model):
            kps = pitch_module.detect_pitch_keypoints(synthetic_frame)
            assert kps == []

    def test_returns_empty_when_no_results(self, synthetic_frame):
        mock_model = MagicMock()
        mock_model.return_value = []

        import pitch as pitch_module

        with patch.object(pitch_module, "get_pitch_model", return_value=mock_model):
            kps = pitch_module.detect_pitch_keypoints(synthetic_frame)
            assert kps == []

    def test_returns_empty_when_keypoints_data_empty(self, synthetic_frame):
        kp_data = np.zeros((0, 32, 3), dtype=np.float32)
        mock_cpu = MagicMock()
        mock_cpu.numpy.return_value = kp_data
        mock_kp = MagicMock()
        mock_kp.data = MagicMock()
        mock_kp.data.cpu.return_value = mock_cpu
        mock_result = MagicMock()
        mock_result.keypoints = mock_kp
        mock_model = MagicMock()
        mock_model.return_value = [mock_result]

        import pitch as pitch_module

        with patch.object(pitch_module, "get_pitch_model", return_value=mock_model):
            from pitch import detect_pitch_keypoints
            kps = detect_pitch_keypoints(synthetic_frame)
            assert kps == []


class TestComputeHomography:
    def test_computes_homography_with_corners(self):
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ] + [(500.0, 500.0, 0.1) for _ in range(28)]

        from pitch import compute_homography
        H = compute_homography(keypoints)
        assert H is not None
        assert H.shape == (3, 3)

    def test_computes_homography_with_field_region(self):
        keypoints = [
            (200.0, 100.0, 0.95),
            (1700.0, 100.0, 0.95),
            (1700.0, 900.0, 0.95),
            (200.0, 900.0, 0.95),
        ] + [(950.0, 500.0, 0.1) for _ in range(28)]

        from pitch import compute_homography
        H = compute_homography(keypoints)
        assert H is not None
        assert H.shape == (3, 3)

    def test_returns_none_with_few_points(self):
        keypoints = [(100.0, 100.0, 0.95), (200.0, 200.0, 0.95)]

        from pitch import compute_homography
        H = compute_homography(keypoints)
        assert H is None

    def test_returns_none_when_all_low_confidence(self):
        keypoints = [(0.0, 0.0, 0.1), (100.0, 0.0, 0.1), (100.0, 100.0, 0.1), (0.0, 100.0, 0.1)]

        from pitch import compute_homography
        H = compute_homography(keypoints)
        assert H is None

    def test_returns_none_with_duplicate_corners(self):
        keypoints = [(500.0, 500.0, 0.95), (500.0, 500.0, 0.95), (500.0, 500.0, 0.95), (500.0, 500.0, 0.95)]

        from pitch import compute_homography
        H = compute_homography(keypoints)
        assert H is None

    def test_returns_none_with_empty_list(self):
        from pitch import compute_homography
        H = compute_homography([])
        assert H is None


class TestPixelToPitch:
    def test_transforms_correctly_with_corner_homography(self):
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ]

        from pitch import compute_homography, pixel_to_pitch
        H = compute_homography(keypoints)
        x, y = pixel_to_pitch(960.0, 540.0, H)
        assert x == pytest.approx(50.0, abs=1.0)
        assert y == pytest.approx(50.0, abs=1.0)

    def test_clamps_to_zero(self):
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ]

        from pitch import compute_homography, pixel_to_pitch
        H = compute_homography(keypoints)
        x, y = pixel_to_pitch(-1000.0, -1000.0, H)
        assert x == 0.0
        assert y == 0.0

    def test_clamps_to_one_hundred(self):
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ]

        from pitch import compute_homography, pixel_to_pitch
        H = compute_homography(keypoints)
        x, y = pixel_to_pitch(10000.0, 10000.0, H)
        assert x == 100.0
        assert y == 100.0

    def test_field_region_maps_correctly(self):
        keypoints = [
            (200.0, 100.0, 0.95),
            (1700.0, 100.0, 0.95),
            (1700.0, 900.0, 0.95),
            (200.0, 900.0, 0.95),
        ]

        from pitch import compute_homography, pixel_to_pitch
        H = compute_homography(keypoints)
        cx = (200.0 + 1700.0) / 2.0
        cy = (100.0 + 900.0) / 2.0
        x, y = pixel_to_pitch(cx, cy, H)
        assert x == pytest.approx(50.0, abs=1.0)
        assert y == pytest.approx(50.0, abs=1.0)


class TestPixelToPitchBatch:
    def test_transforms_multiple_points(self):
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ]

        from pitch import compute_homography, pixel_to_pitch_batch
        H = compute_homography(keypoints)
        points = [(0.0, 0.0), (960.0, 540.0), (1919.0, 1079.0)]
        results = pixel_to_pitch_batch(points, H)

        assert len(results) == 3
        assert results[0][0] == pytest.approx(0.0, abs=1.0)
        assert results[0][1] == pytest.approx(0.0, abs=1.0)
        assert results[1][0] == pytest.approx(50.0, abs=1.0)
        assert results[1][1] == pytest.approx(50.0, abs=1.0)
        assert results[2][0] == pytest.approx(100.0, abs=1.0)
        assert results[2][1] == pytest.approx(100.0, abs=1.0)

    def test_returns_empty_for_empty_input(self):
        import numpy as np
        H = np.eye(3, dtype=np.float64)

        from pitch import pixel_to_pitch_batch
        assert pixel_to_pitch_batch([], H) == []


class TestPitchCoordsWithHomography:
    @pytest.fixture(autouse=True)
    def reset_homography(self):
        from events import set_homography
        yield
        set_homography(None)

    def test_uses_homography_when_set(self):
        keypoints = [
            (200.0, 100.0, 0.95),
            (1700.0, 100.0, 0.95),
            (1700.0, 900.0, 0.95),
            (200.0, 900.0, 0.95),
        ]

        from pitch import compute_homography
        from events import _pitch_coords, set_homography
        H = compute_homography(keypoints)
        set_homography(H)

        coords = _pitch_coords((950.0, 500.0), (1920.0, 1080.0))
        assert coords["x"] == 50.0
        assert coords["y"] == 50.0

    def test_falls_back_to_naive_when_no_homography(self):
        from events import _pitch_coords, set_homography
        set_homography(None)

        coords = _pitch_coords((320.0, 240.0), (1920.0, 1080.0))
        expected_x = round((320.0 / 1920.0) * 100.0, 1)
        expected_y = round((240.0 / 1080.0) * 100.0, 1)
        assert coords == {"x": expected_x, "y": expected_y}

    def test_falls_back_on_homography_exception(self):
        import numpy as np
        from events import _pitch_coords, set_homography

        bad_H = np.array([[1, 2], [3, 4]], dtype=object)
        set_homography(bad_H)

        coords = _pitch_coords((960.0, 540.0), (1920.0, 1080.0))
        assert coords == {"x": 50.0, "y": 50.0}

    def test_differs_from_naive_with_field_region(self):
        keypoints = [
            (200.0, 100.0, 0.95),
            (1700.0, 100.0, 0.95),
            (1700.0, 900.0, 0.95),
            (200.0, 900.0, 0.95),
        ]

        from pitch import compute_homography
        from events import _pitch_coords, set_homography
        H = compute_homography(keypoints)
        set_homography(H)

        homography_coords = _pitch_coords((950.0, 500.0), (1920.0, 1080.0))

        set_homography(None)
        naive_coords = _pitch_coords((950.0, 500.0), (1920.0, 1080.0))

        assert homography_coords != naive_coords
        assert homography_coords == {"x": 50.0, "y": 50.0}


class TestIntegration:
    @pytest.fixture(autouse=True)
    def reset_homography(self):
        from events import set_homography
        yield
        set_homography(None)

    def test_detect_pitch_integration_with_homography(self, synthetic_frame):
        kp_data = _make_corner_keypoints(640, 480)
        mock_result = _make_mock_pitch_result(kp_data)
        mock_model = MagicMock()
        mock_model.return_value = [mock_result]

        import pitch as pitch_module

        with patch.object(pitch_module, "get_pitch_model", return_value=mock_model):
            kps = pitch_module.detect_pitch_keypoints(synthetic_frame)
            assert len(kps) == 32

            H = pitch_module.compute_homography(kps)
            assert H is not None
            assert H.shape == (3, 3)

            x, y = pitch_module.pixel_to_pitch(320.0, 240.0, H)
            assert x == pytest.approx(50.0, abs=1.0)
            assert y == pytest.approx(50.0, abs=1.0)


class TestSetHomography:
    @pytest.fixture(autouse=True)
    def reset_homography(self):
        from events import set_homography
        yield
        set_homography(None)

    def test_set_and_clear_homography(self):
        import numpy as np
        import events
        assert events._HOMOGRAPHY is None

        H = np.eye(3, dtype=np.float64)
        events.set_homography(H)
        assert events._HOMOGRAPHY is not None
        np.testing.assert_array_equal(events._HOMOGRAPHY, H)

        events.set_homography(None)
        assert events._HOMOGRAPHY is None


class TestUnloadModels:
    def test_unload_clears_pitch_model(self):
        import models as m
        m._pitch_model = "something"
        m.unload_models()
        assert m._pitch_model is None

    def test_get_pitch_model_loads_on_first_call(self):
        import models as m
        m._pitch_model = None
        yolo_instance = MagicMock()
        with patch("models.YOLO", return_value=yolo_instance):
            result = m.get_pitch_model()
            assert result is yolo_instance


class TestHeatmapScalePos:
    def test_scale_pos_uses_homography_when_provided(self):
        from heatmap import _scale_pos
        keypoints = [
            (0.0, 0.0, 0.95),
            (1919.0, 0.0, 0.95),
            (1919.0, 1079.0, 0.95),
            (0.0, 1079.0, 0.95),
        ]
        from pitch import compute_homography
        H = compute_homography(keypoints)
        scaled = _scale_pos((960.0, 540.0), H)
        assert scaled[0] > 0
        assert scaled[1] > 0

    def test_scale_pos_falls_back_without_homography(self):
        from heatmap import _scale_pos
        scaled = _scale_pos((960.0, 540.0), None)
        assert scaled[0] > 0
        assert scaled[1] > 0
