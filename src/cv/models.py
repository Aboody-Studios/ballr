from ultralytics import YOLO
import logging

import config

logger = logging.getLogger(__name__)

_detection_model = None
_pose_model = None
_ball_model = None
_pitch_model = None


def get_detection_model() -> YOLO:
    global _detection_model
    if _detection_model is None:
        model_name = config.CV_MODEL_DETECTION
        logger.info("loading detection model: %s", model_name)
        _detection_model = YOLO(model_name)
    return _detection_model


def get_pose_model() -> YOLO:
    global _pose_model
    if _pose_model is None:
        model_name = config.CV_MODEL_POSE
        logger.info("loading pose model: %s", model_name)
        _pose_model = YOLO(model_name)
    return _pose_model


def get_ball_model() -> YOLO:
    global _ball_model
    if _ball_model is None:
        model_name = config.CV_MODEL_BALL
        logger.info("loading ball fallback model: %s", model_name)
        _ball_model = YOLO(model_name)
    return _ball_model


def get_pitch_model() -> YOLO:
    global _pitch_model
    if _pitch_model is None:
        model_name = config.CV_MODEL_PITCH
        logger.info("loading pitch detection model: %s", model_name)
        _pitch_model = YOLO(model_name)
    return _pitch_model


def unload_models() -> None:
    global _detection_model, _pose_model, _ball_model, _pitch_model
    _detection_model = None
    _pose_model = None
    _ball_model = None
    _pitch_model = None
    import gc
    gc.collect()
