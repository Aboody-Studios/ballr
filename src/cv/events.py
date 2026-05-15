import math

import config

logger = __import__("logging").getLogger(__name__)

_FOOT_INDICES = [13, 14, 15, 16]
_FOOT_PROXIMITY_PX = 2.0 * config.CV_PX_PER_METER
_PASS_SPEED_THRESHOLD = 3.0
_SHOT_SPEED_THRESHOLD = 12.0
_SPRINT_SPEED_THRESHOLD = 7.0
_DRIBBLE_PROXIMITY_FRAMES = 3
_TACKLE_SPEED_HIGH = 5.0
_TACKLE_SPEED_LOW = 1.0
_TACKLE_WINDOW = 3


def _min_foot_distance(ball_center: tuple[float, float], keypoints: list) -> float:
    bx, by = ball_center
    min_dist = float("inf")
    for idx in _FOOT_INDICES:
        if idx < len(keypoints):
            kp = keypoints[idx]
            if kp[2] > 0.3:
                dist = math.hypot(bx - kp[0], by - kp[1])
                if dist < min_dist:
                    min_dist = dist
    return min_dist


def _ball_velocity_changes(history: list[dict], i: int, window: int = 3) -> bool:
    if i < window:
        return False
    before = []
    for j in range(i - window, i):
        vel = history[j].get("ball_velocity")
        if vel:
            before.append(vel)
    after = history[i].get("ball_velocity")
    if not before or not after:
        return False
    avg_before = (sum(v[0] for v in before) / len(before),
                  sum(v[1] for v in before) / len(before))
    dot = avg_before[0] * after[0] + avg_before[1] * after[1]
    mag_before = math.hypot(*avg_before)
    mag_after = math.hypot(*after)
    if mag_before < 1 or mag_after < 1:
        return False
    cos_angle = dot / (mag_before * mag_after)
    return cos_angle < -0.3


def _pitch_coords(pixel_center: tuple[float, float], frame_size: tuple[int, int] = (1920, 1080)) -> dict:
    fx, fy = pixel_center
    fw, fh = frame_size
    return {
        "x": round((fx / fw) * 100.0, 1),
        "y": round((fy / fh) * 100.0, 1),
    }


def _make_ts(entry: dict) -> str:
    return entry.get("timestamp", "00:00")


def detect_passes(history: list[dict]) -> list[dict]:
    events = []
    for i in range(1, len(history)):
        h = history[i]
        if h["ball_center"] is None or h["keypoints"] is None:
            continue
        foot_dist = _min_foot_distance(h["ball_center"], h["keypoints"])
        ball_speed = h.get("ball_speed", 0)
        if foot_dist < _FOOT_PROXIMITY_PX and ball_speed > _PASS_SPEED_THRESHOLD:
            vel_change = _ball_velocity_changes(history, i)
            if vel_change:
                coords = _pitch_coords(h["player_center"]) if h.get("player_center") else _pitch_coords(h["ball_center"])
                direction = h.get("ball_velocity", (0, 0))
                result = "SUCCESS" if direction[1] < 0 else "FAILURE"
                events.append({
                    "timestamp": _make_ts(h),
                    "type": "PASS",
                    "result": result,
                    "coordinates": coords,
                    "insight": f"Pass detected at {_make_ts(h)}",
                })
    return events


def detect_shots(history: list[dict]) -> list[dict]:
    events = []
    for i in range(1, len(history)):
        h = history[i]
        if h["ball_center"] is None or h["keypoints"] is None:
            continue
        foot_dist = _min_foot_distance(h["ball_center"], h["keypoints"])
        ball_speed = h.get("ball_speed", 0)
        if foot_dist < _FOOT_PROXIMITY_PX and ball_speed > _SHOT_SPEED_THRESHOLD:
            coords = _pitch_coords(h["player_center"]) if h.get("player_center") else _pitch_coords(h["ball_center"])
            direction = h.get("ball_velocity", (0, 0))
            result = "SUCCESS" if direction[0] > 0 else "FAILURE"
            events.append({
                "timestamp": _make_ts(h),
                "type": "SHOT",
                "result": result,
                "coordinates": coords,
                "insight": f"Shot detected at {_make_ts(h)}, speed={ball_speed:.1f}m/s",
            })
    return events


def detect_dribbles(history: list[dict]) -> list[dict]:
    events = []
    proximity_streak = 0
    streak_start = None
    for i in range(len(history)):
        h = history[i]
        if h["ball_center"] is None or h["keypoints"] is None:
            proximity_streak = 0
            streak_start = None
            continue
        foot_dist = _min_foot_distance(h["ball_center"], h["keypoints"])
        player_speed = h.get("player_speed", 0)
        if foot_dist < _FOOT_PROXIMITY_PX and player_speed > 1.0:
            if proximity_streak == 0:
                streak_start = i
            proximity_streak += 1
        else:
            if proximity_streak >= _DRIBBLE_PROXIMITY_FRAMES and streak_start is not None:
                start_h = history[streak_start]
                coords = _pitch_coords(start_h["player_center"]) if start_h.get("player_center") else {"x": 50.0, "y": 50.0}
                events.append({
                    "timestamp": _make_ts(start_h),
                    "type": "DRIBBLE",
                    "result": "NEUTRAL",
                    "coordinates": coords,
                    "insight": f"Dribble sequence of {proximity_streak} frames",
                })
            proximity_streak = 0
            streak_start = None
    if proximity_streak >= _DRIBBLE_PROXIMITY_FRAMES and streak_start is not None:
        start_h = history[streak_start]
        coords = _pitch_coords(start_h["player_center"]) if start_h.get("player_center") else {"x": 50.0, "y": 50.0}
        events.append({
            "timestamp": _make_ts(start_h),
            "type": "DRIBBLE",
            "result": "NEUTRAL",
            "coordinates": coords,
            "insight": f"Dribble sequence of {proximity_streak} frames",
        })
    return events


def detect_sprints(history: list[dict]) -> list[dict]:
    events = []
    sprint_streak = 0
    streak_start = None
    for i in range(len(history)):
        h = history[i]
        speed = h.get("player_speed", 0)
        if speed > _SPRINT_SPEED_THRESHOLD:
            if sprint_streak == 0:
                streak_start = i
            sprint_streak += 1
        else:
            if sprint_streak > 0 and streak_start is not None:
                start_h = history[streak_start]
                end_h = history[i - 1] if i > 0 else start_h
                distance = sum(
                    history[j].get("player_speed", 0) * (1.0 / config.CV_FPS_TARGET)
                    for j in range(streak_start, i)
                )
                max_speed = max(
                    history[j].get("player_speed", 0)
                    for j in range(streak_start, i)
                )
                coords = _pitch_coords(start_h["player_center"]) if start_h.get("player_center") else {"x": 50.0, "y": 50.0}
                events.append({
                    "timestamp": _make_ts(start_h),
                    "type": "SPRINT",
                    "result": "NEUTRAL",
                    "coordinates": coords,
                    "insight": f"Sprint for {sprint_streak} frames, max speed {max_speed * 3.6:.1f}km/h, distance {distance:.1f}m",
                })
            sprint_streak = 0
            streak_start = None
    if sprint_streak > 0 and streak_start is not None:
        start_h = history[streak_start]
        coords = _pitch_coords(start_h["player_center"]) if start_h.get("player_center") else {"x": 50.0, "y": 50.0}
        events.append({
            "timestamp": _make_ts(start_h),
            "type": "SPRINT",
            "result": "NEUTRAL",
            "coordinates": coords,
            "insight": f"Sprint for {sprint_streak} frames",
        })
    return events


def detect_tackles(history: list[dict]) -> list[dict]:
    events = []
    for i in range(2, len(history)):
        h = history[i]
        if h["ball_center"] is None:
            continue
        ball_near = _min_foot_distance(h["ball_center"], h["keypoints"]) if h["keypoints"] else float("inf")
        if ball_near > _FOOT_PROXIMITY_PX * 2:
            continue
        prev_speeds = []
        for j in range(max(0, i - _TACKLE_WINDOW), i + 1):
            prev_speeds.append(history[j].get("player_speed", 0))
        if not prev_speeds:
            continue
        recent_avg = sum(prev_speeds[:-1]) / len(prev_speeds[:-1]) if len(prev_speeds) > 1 else 0
        current = prev_speeds[-1]
        if recent_avg > _TACKLE_SPEED_HIGH and current < _TACKLE_SPEED_LOW and ball_near < _FOOT_PROXIMITY_PX * 2:
            coords = _pitch_coords(h["player_center"]) if h.get("player_center") else _pitch_coords(h["ball_center"])
            events.append({
                "timestamp": _make_ts(h),
                "type": "TACKLE",
                "result": "NEUTRAL",
                "coordinates": coords,
                "insight": f"Tackle detected at {_make_ts(h)}",
            })
    return events


def detect_scanning(history: list[dict]) -> list[dict]:
    events = []
    prev_orientation = None
    for i in range(len(history)):
        h = history[i]
        orientation = h.get("head_orientation", "unknown")
        if orientation != "unknown" and prev_orientation is not None and prev_orientation != "unknown":
            if orientation != prev_orientation:
                coords = _pitch_coords(h["player_center"]) if h.get("player_center") else {"x": 50.0, "y": 50.0}
                events.append({
                    "timestamp": _make_ts(h),
                    "type": "RECOVERY",
                    "result": "NEUTRAL",
                    "coordinates": coords,
                    "insight": f"Scanning: head turned {orientation}",
                })
        prev_orientation = orientation
    return events


def extract_events(history: list[dict]) -> list[dict]:
    passes = detect_passes(history)
    shots = detect_shots(history)
    dribbles = detect_dribbles(history)
    sprints = detect_sprints(history)
    tackles = detect_tackles(history)
    scanning = detect_scanning(history)
    all_events = passes + shots + dribbles + sprints + tackles + scanning
    all_events.sort(key=lambda e: e["timestamp"])
    return all_events


def compute_summary(history: list[dict], events: list[dict]) -> dict:
    total_distance_m = 0.0
    top_speed_ms = 0.0
    for h in history:
        speed = h.get("player_speed", 0)
        dt = 1.0 / config.CV_FPS_TARGET
        total_distance_m += speed * dt
        if speed > top_speed_ms:
            top_speed_ms = speed
    passes = [e for e in events if e["type"] == "PASS"]
    successful_passes = len([e for e in passes if e["result"] == "SUCCESS"])
    pass_accuracy = successful_passes / len(passes) if passes else 0.0
    sprints = len([e for e in events if e["type"] == "SPRINT"])
    touches = 0
    for e in events:
        if e["type"] == "DRIBBLE":
            touches += 2
    return {
        "total_distance": round(total_distance_m / 1000.0, 2),
        "top_speed": round(top_speed_ms * 3.6, 1),
        "pass_accuracy": round(pass_accuracy, 2),
        "touches": touches,
        "sprints": sprints,
    }
