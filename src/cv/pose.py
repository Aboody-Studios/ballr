import logging
from typing import Optional

import numpy as np

from models import get_pose_model

logger = logging.getLogger(__name__)

COCO_KEYPOINT_NAMES = [
    "nose", "left_eye", "right_eye", "left_ear", "right_ear",
    "left_shoulder", "right_shoulder", "left_elbow", "right_elbow",
    "left_wrist", "right_wrist", "left_hip", "right_hip",
    "left_knee", "right_knee", "left_ankle", "right_ankle",
]

_FOOT_INDICES = [15, 16, 13, 14]
_SHOULDER_INDICES = [5, 6]
_HIP_INDICES = [11, 12]


def _compute_iou(bbox_a: tuple, bbox_b: tuple) -> float:
    ax1, ay1, ax2, ay2 = bbox_a
    bx1, by1, bx2, by2 = bbox_b
    xi1 = max(ax1, bx1)
    yi1 = max(ay1, by1)
    xi2 = min(ax2, bx2)
    yi2 = min(ay2, by2)
    inter = max(0, xi2 - xi1) * max(0, yi2 - yi1)
    area_a = (ax2 - ax1) * (ay2 - ay1)
    area_b = (bx2 - bx1) * (by2 - by1)
    union = area_a + area_b - inter
    if union <= 0:
        return 0.0
    return inter / union


def _parse_pose_keypoints(result) -> list[list[tuple[float, float, float]]]:
    if not hasattr(result, "keypoints") or result.keypoints is None:
        return []
    kps_data = result.keypoints.data
    if kps_data is None:
        return []
    kps_np = kps_data.cpu().numpy()
    people = []
    for person in kps_np:
        keypoints = [(float(kp[0]), float(kp[1]), float(kp[2])) for kp in person]
        people.append(keypoints)
    return people


def estimate_pose(frame: np.ndarray, bbox: tuple, conf_threshold: float = 0.5) -> Optional[dict]:
    model = get_pose_model()
    results = model(frame, verbose=False, conf=conf_threshold)
    if not results or len(results) == 0:
        return None
    all_people = _parse_pose_keypoints(results[0])
    if not all_people:
        return None
    if len(all_people) == 1:
        kps = all_people[0]
        avg_conf = np.mean([k[2] for k in kps])
        return {"keypoints": kps, "confidence": float(avg_conf)}
    person_bboxes = []
    for kps in all_people:
        xs = [k[0] for k in kps if k[2] > 0]
        ys = [k[1] for k in kps if k[2] > 0]
        if xs and ys:
            p_bbox = (min(xs), min(ys), max(xs), max(ys))
        else:
            p_bbox = bbox
        person_bboxes.append(p_bbox)
    overlaps = [_compute_iou(bbox, pb) for pb in person_bboxes]
    best_idx = int(np.argmax(overlaps))
    if overlaps[best_idx] <= 0:
        best_idx = 0
    kps = all_people[best_idx]
    avg_conf = np.mean([k[2] for k in kps])
    return {"keypoints": kps, "confidence": float(avg_conf)}


def compute_angle(a: tuple, b: tuple, c: tuple) -> float:
    ba = np.array([a[0] - b[0], a[1] - b[1]])
    bc = np.array([c[0] - b[0], c[1] - b[1]])
    dot = float(np.dot(ba, bc))
    norm_ba = float(np.linalg.norm(ba))
    norm_bc = float(np.linalg.norm(bc))
    if norm_ba < 1e-6 or norm_bc < 1e-6:
        return 0.0
    cos_angle = max(-1.0, min(1.0, dot / (norm_ba * norm_bc)))
    return float(np.degrees(np.arccos(cos_angle)))


def compute_joint_angles(keypoints: list) -> dict:
    kp = {name: (keypoints[i][0], keypoints[i][1])
          for i, name in enumerate(COCO_KEYPOINT_NAMES)
          if i < len(keypoints) and keypoints[i][2] > 0.3}
    angles = {}
    angle_defs = [
        ("left_knee", "left_hip", "left_knee", "left_ankle"),
        ("right_knee", "right_hip", "right_knee", "right_ankle"),
        ("left_hip", "left_shoulder", "left_hip", "left_knee"),
        ("right_hip", "right_shoulder", "right_hip", "right_knee"),
        ("left_shoulder", "left_elbow", "left_shoulder", "left_hip"),
        ("right_shoulder", "right_elbow", "right_shoulder", "right_hip"),
        ("left_elbow", "left_shoulder", "left_elbow", "left_wrist"),
        ("right_elbow", "right_shoulder", "right_elbow", "right_wrist"),
        ("left_ankle", "left_knee", "left_ankle", "left_hip"),
        ("right_ankle", "right_knee", "right_ankle", "right_hip"),
    ]
    for name, a_name, b_name, c_name in angle_defs:
        if a_name in kp and b_name in kp and c_name in kp:
            angles[name] = round(compute_angle(kp[a_name], kp[b_name], kp[c_name]), 1)
    return angles


def compute_head_orientation(keypoints: list) -> str:
    if len(keypoints) < 17:
        return "unknown"
    nose = keypoints[0]
    ls = keypoints[5]
    rs = keypoints[6]
    if nose[2] < 0.3 or ls[2] < 0.3 or rs[2] < 0.3:
        return "unknown"
    nose_x = nose[0]
    shoulder_mid_x = (ls[0] + rs[0]) / 2.0
    diff = nose_x - shoulder_mid_x
    if abs(diff) < 10:
        return "center"
    if diff > 0:
        return "right"
    return "left"
