import logging
import os

import cv2

import config
from detector import detect_all, select_target_player
from events import extract_events, compute_summary
from heatmap import generate_all_heatmaps, upload_tracking_data
from events import set_homography
from models import get_detection_model, get_pose_model, get_pitch_model, unload_models
from pitch import compute_homography, detect_pitch_keypoints
from pose import compute_head_orientation, compute_joint_angles, estimate_pose
from tracker import TrackingState

logger = logging.getLogger(__name__)


def _format_timestamp(frame_idx: int, fps: float) -> str:
    total_sec = int(frame_idx / fps) if fps > 0 else 0
    minutes = total_sec // 60
    seconds = total_sec % 60
    return f"{minutes:02d}:{seconds:02d}"


def run_pipeline(
    video_path: str,
    match_id: str,
    user_id: str,
    shirt_number: int,
    position: str,
) -> dict:
    if not os.path.isfile(video_path):
        raise FileNotFoundError(f"video file not found: {video_path}")

    cap = cv2.VideoCapture(video_path)
    if not cap.isOpened():
        raise ValueError(f"could not open video: {video_path}")

    try:
        original_fps = cap.get(cv2.CAP_PROP_FPS)
        if original_fps <= 0:
            original_fps = 30.0

        total_frames = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        if total_frames < config.CV_MIN_FRAMES:
            raise ValueError(f"video too short: {total_frames} frames, minimum {config.CV_MIN_FRAMES} required")

        frame_width = int(cap.get(cv2.CAP_PROP_FRAME_WIDTH))
        frame_height = int(cap.get(cv2.CAP_PROP_FRAME_HEIGHT))
        frame_center = (frame_width / 2.0, frame_height / 2.0)

        skip_interval = max(1, int(original_fps / config.CV_FPS_TARGET))
        target_fps = original_fps / skip_interval

        logger.info(
            "video: %s | %dx%d | %.1f fps | total frames: %d | skip: %d | target fps: %.1f",
            video_path, frame_width, frame_height, original_fps, total_frames, skip_interval, target_fps,
        )

        get_detection_model()
        get_pose_model()
        get_pitch_model()

        tracker = TrackingState(
            process_noise=config.CV_KALMAN_PROCESS_NOISE,
            measurement_noise=config.CV_KALMAN_MEASUREMENT_NOISE,
            dt=1.0,
        )

        frame_idx = 0
        consecutive_misses = 0
        processed_count = 0

        pitch_interval = max(1, int(original_fps * 50))
        homography = None

        while True:
            ret, frame = cap.read()
            if not ret:
                break

            if frame_idx % skip_interval != 0:
                frame_idx += 1
                continue

            frame = cv2.cvtColor(frame, cv2.COLOR_BGR2RGB)

            if frame_idx % pitch_interval == 0:
                try:
                    pitch_kps = detect_pitch_keypoints(frame)
                    H = compute_homography(pitch_kps)
                    if H is not None:
                        homography = H
                        set_homography(H)
                        logger.info("homography computed from pitch keypoints on frame %d", frame_idx)
                    else:
                        logger.warning("could not compute homography on frame %d, using naive fallback", frame_idx)
                except Exception:
                    logger.exception("pitch detection failed on frame %d", frame_idx)
            timestamp = _format_timestamp(frame_idx, original_fps)

            player_detections, ball_detection = detect_all(frame, conf_threshold=config.CV_DETECTION_CONF)

            if shirt_number > 0 and config.CV_ENABLE_OCR:
                from ocr import find_best_player_by_shirt

                target = find_best_player_by_shirt(player_detections, frame, shirt_number)
            else:
                target = select_target_player(player_detections, frame_center)

            keypoints = None
            joint_angles = None
            head_orientation = None
            if target is not None:
                player_bbox = target["bbox"]
                player_center = target["center"]
                pose_result = estimate_pose(frame, player_bbox, conf_threshold=config.CV_POSE_CONF)
                if pose_result:
                    keypoints = pose_result["keypoints"]
                    joint_angles = compute_joint_angles(keypoints)
                    head_orientation = compute_head_orientation(keypoints)
                consecutive_misses = 0
            else:
                player_center = None
                consecutive_misses += 1
                if consecutive_misses > 100:
                    logger.warning("no player detection for %d frames (frame %d)", consecutive_misses, frame_idx)

            ball_center = ball_detection["center"] if ball_detection else None

            tracker.update(
                frame_idx=frame_idx,
                timestamp=timestamp,
                keypoints=keypoints,
                ball_center=ball_center,
                player_center=player_center,
                joint_angles=joint_angles,
                head_orientation=head_orientation,
                px_per_meter=config.CV_PX_PER_METER,
                skip_interval=skip_interval,
                target_fps=config.CV_FPS_TARGET,
            )

            processed_count += 1
            frame_idx += 1

    finally:
        cap.release()
        unload_models()

    logger.info("processed %d frames", processed_count)

    history = tracker.get_history()
    if not history or not any(h.get("player_center") for h in history):
        logger.warning("no player detections in any frame, returning empty result")
        return {
            "match_id": match_id,
            "summary": {
                "total_distance": 0.0,
                "top_speed": 0.0,
                "pass_accuracy": 0.0,
                "touches": 0,
                "sprints": 0,
            },
            "heatmaps": {"overall_url": "", "defensive_url": "", "attacking_url": ""},
            "events": [],
            "tracking_data_url": "",
        }

    events = extract_events(history)
    summary = compute_summary(history, events)

    output_dir = config.CV_HEATMAP_OUTPUT_DIR
    bucket = config.CV_S3_BUCKET

    heatmaps = generate_all_heatmaps(history, match_id, output_dir, bucket, homography=homography)

    tracking_url = upload_tracking_data(history, match_id, bucket) if history else ""

    return {
        "match_id": match_id,
        "summary": summary,
        "heatmaps": heatmaps,
        "events": events,
        "tracking_data_url": tracking_url,
    }
