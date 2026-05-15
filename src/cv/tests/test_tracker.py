import numpy as np
import pytest

from tracker import KalmanFilter1D, KalmanFilter2D, TrackingState


class TestKalmanFilter1D:
    def test_update_returns_close_to_measurement(self):
        kf = KalmanFilter1D()
        result = kf.update(50.0)
        assert result == pytest.approx(50.0, abs=15.0)

    def test_predict_returns_reasonable_value(self):
        kf = KalmanFilter1D()
        kf.update(100.0)
        pred = kf.predict()
        assert isinstance(pred, float)
        assert not np.isnan(pred)

    def test_repeated_updates_converge(self):
        kf = KalmanFilter1D(process_noise=1e-6, measurement_noise=1e-4)
        for _ in range(20):
            kf.update(42.0)
        result = kf.update(42.0)
        assert result == pytest.approx(42.0, abs=0.5)

    def test_converges_to_constant_value(self):
        kf = KalmanFilter1D(process_noise=1e-6, measurement_noise=1e-2)
        kf.update(100.0)
        for _ in range(15):
            kf.update(100.0)
        last = kf.update(100.0)
        assert last == pytest.approx(100.0, abs=1.0)

    def test_velocity_property(self):
        kf = KalmanFilter1D()
        kf.update(100.0)
        vel = kf.velocity
        assert isinstance(vel, float)


class TestKalmanFilter2D:
    def test_update_returns_close_to_measurement(self):
        kf = KalmanFilter2D()
        rx, ry = kf.update(100.0, 200.0)
        assert rx == pytest.approx(100.0, abs=15.0)
        assert ry == pytest.approx(200.0, abs=15.0)

    def test_predict_returns_reasonable_value(self):
        kf = KalmanFilter2D()
        kf.update(100.0, 200.0)
        px, py = kf.predict()
        assert isinstance(px, float)
        assert isinstance(py, float)

    def test_velocity_properties(self):
        kf = KalmanFilter2D()
        kf.update(100.0, 200.0)
        assert isinstance(kf.velocity_x, float)
        assert isinstance(kf.velocity_y, float)
        assert isinstance(kf.speed, float)
        assert kf.speed >= 0


class TestTrackingState:
    def test_update_appends_to_history(self):
        ts = TrackingState()
        entry = ts.update(
            frame_idx=0,
            timestamp="00:00",
            keypoints=[(100, 200, 0.9) for _ in range(17)],
            ball_center=(300, 200),
            player_center=(320, 240),
            joint_angles={"left_knee": 90.0},
            head_orientation="center",
        )
        assert "frame_idx" in entry
        assert entry["frame_idx"] == 0

    def test_get_history_returns_all_frames(self):
        ts = TrackingState()
        for i in range(5):
            ts.update(
                frame_idx=i,
                timestamp=f"00:0{i}",
                keypoints=[(100 + i, 200, 0.9) for _ in range(17)],
                ball_center=(300 + i, 200),
                player_center=(320 + i, 240),
                joint_angles={"left_knee": 90.0 + i},
                head_orientation="center",
            )
        history = ts.get_history()
        assert len(history) == 5
        for i, h in enumerate(history):
            assert h["frame_idx"] == i

    def test_multiple_updates_no_error(self):
        ts = TrackingState()
        for i in range(100):
            ts.update(
                frame_idx=i,
                timestamp=f"00:{i:02d}",
                keypoints=[(100, 200, 0.9) for _ in range(17)],
                ball_center=(300, 200),
                player_center=(320, 240),
                joint_angles={"left_knee": 90.0},
                head_orientation="center",
            )
        assert len(ts.get_history()) == 100

    def test_get_smoothed_keypoints(self):
        ts = TrackingState()
        kps = [(float(i * 10), float(i * 10 + 5), 0.9) for i in range(17)]
        ts.update(
            frame_idx=0, timestamp="00:00",
            keypoints=kps, ball_center=(300, 200),
            player_center=(320, 240),
            joint_angles={}, head_orientation="unknown",
        )
        smoothed = ts.get_smoothed_keypoints()
        assert smoothed is not None
        assert len(smoothed) == 17

    def test_get_predicted_keypoints(self):
        ts = TrackingState()
        predicted = ts.get_predicted_keypoints()
        assert len(predicted) == 17
        assert all(k[2] == 0.0 for k in predicted)

    def test_update_with_none_values(self):
        ts = TrackingState()
        entry = ts.update(
            frame_idx=0, timestamp="00:00",
            keypoints=None, ball_center=None,
            player_center=None, joint_angles=None,
            head_orientation=None,
        )
        assert entry["player_center"] is not None
        assert entry["ball_center"] is not None
        assert entry["joint_angles"] == {}
        assert entry["head_orientation"] == "unknown"
