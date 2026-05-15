import logging
from typing import Optional

import numpy as np
from models import get_ball_model, get_detection_model

logger = logging.getLogger(__name__)

_PLAYER_CLASSES = {"player", "goalkeeper"}


def _parse_yolo_boxes(result) -> list[dict]:
    boxes_data = result.boxes
    if boxes_data is None or len(boxes_data) == 0:
        return []
    names = result.names if hasattr(result, "names") and result.names else {}
    detections = []
    for i in range(len(boxes_data)):
        xyxy = boxes_data.xyxy[i].tolist()
        conf = float(boxes_data.conf[i])
        cls_id = int(boxes_data.cls[i])
        cls_name = names.get(cls_id, f"class_{cls_id}")
        x1, y1, x2, y2 = xyxy
        cx = (x1 + x2) / 2.0
        cy = (y1 + y2) / 2.0
        detections.append(
            {
                "bbox": (x1, y1, x2, y2),
                "confidence": conf,
                "class": cls_name,
                "center": (cx, cy),
            }
        )
    return detections


def detect_players(frame: np.ndarray, conf_threshold: float = 0.5) -> list[dict]:
    model = get_detection_model()
    results = model(frame, verbose=False)
    if not results or len(results) == 0:
        return []
    all_dets = _parse_yolo_boxes(results[0])
    players = [
        d
        for d in all_dets
        if d["class"] in _PLAYER_CLASSES and d["confidence"] >= conf_threshold
    ]
    return players


def detect_ball(frame: np.ndarray, conf_threshold: float = 0.5) -> Optional[dict]:
    model = get_detection_model()
    results = model(frame, verbose=False)
    if results and len(results) > 0:
        all_dets = _parse_yolo_boxes(results[0])
        for d in all_dets:
            if d["class"] == "ball" and d["confidence"] >= conf_threshold:
                return d
    try:
        fallback = get_ball_model()
        fb_results = fallback(frame, verbose=False)
        if fb_results and len(fb_results) > 0:
            fb_dets = _parse_yolo_boxes(fb_results[0])
            if fb_dets:
                best = max(fb_dets, key=lambda x: x["confidence"])
                if best["confidence"] >= conf_threshold:
                    return best
    except Exception:
        logger.debug("ball fallback model unavailable", exc_info=True)
    return None


def select_target_player(
    detections: list[dict], frame_center: Optional[tuple] = None
) -> Optional[dict]:
    if not detections:
        return None
    if len(detections) == 1:
        return detections[0]

    scores = []
    for d in detections:
        cx, cy = d["center"]
        conf = d["confidence"]
        if frame_center:
            dcx, dcy = frame_center
            dist = np.hypot(cx - dcx, cy - dcy)
            score = conf / (1.0 + dist * 0.001)
        else:
            score = conf
        scores.append((score, d))
    scores.sort(key=lambda x: x[0], reverse=True)
    return scores[0][1]


def crop_player(frame: np.ndarray, bbox: tuple) -> np.ndarray:
    x1, y1, x2, y2 = map(int, bbox)
    h, w = frame.shape[:2]
    x1 = max(0, x1)
    y1 = max(0, y1)
    x2 = min(w, x2)
    y2 = min(h, y2)
    if x2 <= x1 or y2 <= y1:
        return frame[
            max(0, y1 - 20) : min(h, y1 + 20), max(0, x1 - 20) : min(w, x1 + 20)
        ]
    return frame[y1:y2, x1:x2]
