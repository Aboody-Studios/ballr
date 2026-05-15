import math
from typing import Optional

import numpy as np


class KalmanFilter1D:
    def __init__(self, process_noise: float = 1e-4, measurement_noise: float = 1e-2, dt: float = 1.0):
        self.dt = dt
        self.F = np.array([[1.0, dt], [0.0, 1.0]])
        self.H = np.array([[1.0, 0.0]])
        self.Q = np.array([
            [process_noise * dt**3 / 3.0, process_noise * dt**2 / 2.0],
            [process_noise * dt**2 / 2.0, process_noise * dt],
        ])
        self.R = np.array([[measurement_noise]])
        self.x = np.zeros((2, 1))
        self.P = np.eye(2) * 10.0

    def update(self, measurement: Optional[float]) -> float:
        self.x = self.F @ self.x
        self.P = self.F @ self.P @ self.F.T + self.Q
        if measurement is not None:
            y = np.array([[measurement]]) - self.H @ self.x
            S = self.H @ self.P @ self.H.T + self.R
            K = self.P @ self.H.T @ np.linalg.inv(S)
            self.x = self.x + K @ y
            self.P = (np.eye(2) - K @ self.H) @ self.P
        return float(self.x[0, 0])

    def predict(self) -> float:
        self.x = self.F @ self.x
        self.P = self.F @ self.P @ self.F.T + self.Q
        return float(self.x[0, 0])

    @property
    def velocity(self) -> float:
        return float(self.x[1, 0])


class KalmanFilter2D:
    def __init__(self, process_noise: float = 1e-4, measurement_noise: float = 1e-2, dt: float = 1.0):
        self.dt = dt
        self.F = np.array([
            [1.0, 0.0, dt, 0.0],
            [0.0, 1.0, 0.0, dt],
            [0.0, 0.0, 1.0, 0.0],
            [0.0, 0.0, 0.0, 1.0],
        ])
        self.H = np.array([
            [1.0, 0.0, 0.0, 0.0],
            [0.0, 1.0, 0.0, 0.0],
        ])
        q = process_noise
        dt2 = dt**2 / 2.0
        dt3 = dt**3 / 3.0
        self.Q = np.array([
            [dt3, 0.0, dt2, 0.0],
            [0.0, dt3, 0.0, dt2],
            [dt2, 0.0, dt, 0.0],
            [0.0, dt2, 0.0, dt],
        ]) * q
        self.R = np.eye(2) * measurement_noise
        self.x = np.zeros((4, 1))
        self.P = np.eye(4) * 10.0

    def update(self, mx: Optional[float], my: Optional[float]) -> tuple[float, float]:
        self.x = self.F @ self.x
        self.P = self.F @ self.P @ self.F.T + self.Q
        if mx is not None and my is not None:
            z = np.array([[mx], [my]])
            y = z - self.H @ self.x
            S = self.H @ self.P @ self.H.T + self.R
            K = self.P @ self.H.T @ np.linalg.inv(S)
            self.x = self.x + K @ y
            self.P = (np.eye(4) - K @ self.H) @ self.P
        return float(self.x[0, 0]), float(self.x[1, 0])

    def predict(self) -> tuple[float, float]:
        self.x = self.F @ self.x
        self.P = self.F @ self.P @ self.F.T + self.Q
        return float(self.x[0, 0]), float(self.x[1, 0])

    @property
    def velocity_x(self) -> float:
        return float(self.x[2, 0])

    @property
    def velocity_y(self) -> float:
        return float(self.x[3, 0])

    @property
    def speed(self) -> float:
        return math.hypot(self.velocity_x, self.velocity_y)


class TrackingState:
    NUM_KEYPOINTS = 17

    def __init__(self, process_noise: float = 1e-4, measurement_noise: float = 1e-2, dt: float = 1.0):
        self._keypoint_filters = [
            (KalmanFilter1D(process_noise, measurement_noise, dt),
             KalmanFilter1D(process_noise, measurement_noise, dt))
            for _ in range(self.NUM_KEYPOINTS)
        ]
        self._ball_filter = KalmanFilter2D(process_noise, measurement_noise, dt)
        self._player_filter = KalmanFilter2D(process_noise * 10, measurement_noise * 10, dt)
        self._history: list[dict] = []
        self._prev_player_center: Optional[tuple[float, float]] = None
        self._prev_ball_center: Optional[tuple[float, float]] = None
        self._prev_timestamp: Optional[str] = None

    def update(
        self,
        frame_idx: int,
        timestamp: str,
        keypoints: Optional[list[tuple[float, float, float]]],
        ball_center: Optional[tuple[float, float]],
        player_center: Optional[tuple[float, float]],
        joint_angles: Optional[dict],
        head_orientation: Optional[str],
        px_per_meter: float = 15.0,
        skip_interval: int = 1,
        target_fps: int = 10,
    ) -> dict:
        smoothed_kpts = self._update_keypoints(keypoints)
        smoothed_ball = self._update_ball(ball_center)
        smoothed_player = self._update_player(player_center)

        speed = 0.0
        accel = 0.0
        if self._prev_player_center is not None and player_center is not None:
            dt_real = skip_interval / target_fps if target_fps > 0 else 1.0
            dx = player_center[0] - self._prev_player_center[0]
            dy = player_center[1] - self._prev_player_center[1]
            px_dist = math.hypot(dx, dy)
            speed = px_dist / px_per_meter / dt_real if dt_real > 0 else 0.0
            if "player_speed" in self._history[-1] if self._history else False:
                prev_speed = self._history[-1].get("player_speed", 0.0)
                accel = (speed - prev_speed) / dt_real if dt_real > 0 else 0.0

        ball_speed = 0.0
        ball_vel = None
        if self._prev_ball_center is not None and ball_center is not None:
            dx = ball_center[0] - self._prev_ball_center[0]
            dy = ball_center[1] - self._prev_ball_center[1]
            ball_px_dist = math.hypot(dx, dy)
            dt_real = skip_interval / target_fps if target_fps > 0 else 1.0
            ball_speed = ball_px_dist / px_per_meter / dt_real if dt_real > 0 else 0.0
            ball_vel = (dx, dy)

        if player_center:
            self._prev_player_center = player_center
        if ball_center:
            self._prev_ball_center = ball_center
        self._prev_timestamp = timestamp

        entry = {
            "frame_idx": frame_idx,
            "timestamp": timestamp,
            "player_center": smoothed_player,
            "ball_center": smoothed_ball,
            "keypoints": smoothed_kpts,
            "joint_angles": joint_angles or {},
            "head_orientation": head_orientation or "unknown",
            "player_speed": round(speed, 4),
            "player_acceleration": round(accel, 4),
            "ball_speed": round(ball_speed, 4),
            "ball_velocity": ball_vel,
        }
        self._history.append(entry)
        return entry

    def _update_keypoints(self, keypoints: Optional[list]) -> Optional[list[tuple[float, float, float]]]:
        if keypoints is None:
            smoothed = []
            for xf, yf in self._keypoint_filters:
                sx = xf.predict()
                sy = yf.predict()
                smoothed.append((sx, sy, 0.0))
            return smoothed

        smoothed = []
        for i in range(self.NUM_KEYPOINTS):
            xf, yf = self._keypoint_filters[i]
            if i < len(keypoints):
                x, y, conf = keypoints[i]
                if conf > 0:
                    sx = xf.update(x)
                    sy = yf.update(y)
                else:
                    sx = xf.predict()
                    sy = yf.predict()
                smoothed.append((sx, sy, conf))
            else:
                sx = xf.predict()
                sy = yf.predict()
                smoothed.append((sx, sy, 0.0))
        return smoothed

    def _update_ball(self, ball_center: Optional[tuple[float, float]]) -> Optional[tuple[float, float]]:
        if ball_center is not None:
            return self._ball_filter.update(ball_center[0], ball_center[1])
        return self._ball_filter.predict()

    def _update_player(self, player_center: Optional[tuple[float, float]]) -> Optional[tuple[float, float]]:
        if player_center is not None:
            return self._player_filter.update(player_center[0], player_center[1])
        return self._player_filter.predict()

    def get_smoothed_keypoints(self) -> Optional[list[tuple[float, float, float]]]:
        if not self._history:
            return None
        return self._history[-1].get("keypoints")

    def get_predicted_keypoints(self) -> list[tuple[float, float, float]]:
        smoothed = []
        for xf, yf in self._keypoint_filters:
            sx = xf.predict()
            sy = yf.predict()
            smoothed.append((sx, sy, 0.0))
        return smoothed

    def get_history(self) -> list[dict]:
        return self._history
