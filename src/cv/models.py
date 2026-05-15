from ultralytics import YOLO
import logging

logger = logging.getLogger(__name__)

_detection_model = None
_pose_model = None
_ball_model = None


def get_detection_model() -> YOLO:
    global _detection_model
    if _detection_model is None:
        logger.info("loading detection model")
        _detection_model = YOLO("Adit-jain/soccana")
    return _detection_model


def get_pose_model() -> YOLO:
    global _pose_model
    if _pose_model is None:
        logger.info("loading pose model")
        _pose_model = YOLO("openvision/yolo26m-pose")
    return _pose_model


def get_ball_model() -> YOLO:
    global _ball_model
    if _ball_model is None:
        logger.info("loading ball fallback model")
        _ball_model = YOLO("martinjolif/yolo-football-ball-detection")
    return _ball_model


def unload_models() -> None:
    global _detection_model, _pose_model, _ball_model
    _detection_model = None
    _pose_model = None
    _ball_model = None
    import gc
    gc.collect()
