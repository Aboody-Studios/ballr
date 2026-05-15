import logging
from typing import Optional

import cv2
import numpy as np

from models import get_pitch_model

logger = logging.getLogger(__name__)


def detect_pitch_keypoints(
    frame: np.ndarray, conf_threshold: float = 0.3
) -> list[tuple[float, float, float]]:
    model = get_pitch_model()
    results = model(frame)

    if not results:
        return []

    result = results[0]
    if result.keypoints is None:
        return []

    kp_tensor = result.keypoints.data
    if kp_tensor is None:
        return []

    kp_np = kp_tensor.cpu().numpy()
    if kp_np.shape[0] == 0:
        return []

    kps = kp_np[0]
    return [(float(x), float(y), float(c)) for x, y, c in kps]


def compute_homography(
    keypoints: list, conf_threshold: float = 0.3
) -> Optional[np.ndarray]:
    high_conf = [(x, y) for x, y, c in keypoints if c >= conf_threshold]

    if len(high_conf) < 4:
        return None

    xs = np.array([p[0] for p in high_conf])
    ys = np.array([p[1] for p in high_conf])

    bbox_corners = [
        (xs.min(), ys.min()),
        (xs.max(), ys.min()),
        (xs.max(), ys.max()),
        (xs.min(), ys.max()),
    ]

    src_pts = []
    used_indices = set()
    for cx, cy in bbox_corners:
        best_idx = -1
        best_dist = float("inf")
        for i, (px, py) in enumerate(high_conf):
            if i in used_indices:
                continue
            dist = np.hypot(px - cx, py - cy)
            if dist < best_dist:
                best_dist = dist
                best_idx = i
        if best_idx >= 0:
            src_pts.append(high_conf[best_idx])
            used_indices.add(best_idx)

    if len(src_pts) < 4:
        return None

    dst_pts = [(0.0, 0.0), (100.0, 0.0), (100.0, 100.0), (0.0, 100.0)]

    src_np = np.array(src_pts, dtype=np.float32)
    dst_np = np.array(dst_pts, dtype=np.float32)

    H, _ = cv2.findHomography(src_np, dst_np, cv2.RANSAC, 5.0)
    return H


def pixel_to_pitch(
    px: float, py: float, homography: np.ndarray
) -> tuple[float, float]:
    pts = np.array([[[px, py]]], dtype=np.float32)
    transformed = cv2.perspectiveTransform(pts, homography)
    x_pct = float(transformed[0, 0, 0])
    y_pct = float(transformed[0, 0, 1])
    x_pct = max(0.0, min(100.0, x_pct))
    y_pct = max(0.0, min(100.0, y_pct))
    return (x_pct, y_pct)


def pixel_to_pitch_batch(
    points: list[tuple[float, float]], homography: np.ndarray
) -> list[tuple[float, float]]:
    if not points:
        return []

    pts = np.array([[[p[0], p[1]]] for p in points], dtype=np.float32)
    transformed = cv2.perspectiveTransform(pts, homography)
    results = []
    for i in range(len(points)):
        x_pct = max(0.0, min(100.0, float(transformed[i, 0, 0])))
        y_pct = max(0.0, min(100.0, float(transformed[i, 0, 1])))
        results.append((x_pct, y_pct))
    return results
