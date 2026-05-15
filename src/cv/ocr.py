"""
Optional shirt number verification via OCR.

Uses PaddleOCR to read shirt numbers from player crops.
Gracefully degrades if PaddleOCR is not available.
"""

import logging
from typing import Optional

import numpy as np
from detector import crop_player

logger = logging.getLogger(__name__)

_ocr = None


def get_ocr():
    global _ocr
    if _ocr is not None:
        return _ocr
    try:
        from paddleocr import PaddleOCR

        _ocr = PaddleOCR(use_angle_cls=False, lang='en', show_log=False, use_gpu=False)
        logger.info("PaddleOCR initialized successfully")
    except ImportError:
        logger.warning("PaddleOCR not installed, OCR disabled")
        _ocr = None
    except OSError:
        logger.warning("PaddleOCR missing system dependencies, OCR disabled")
        _ocr = None
    except Exception:
        logger.exception("Failed to initialize PaddleOCR")
        _ocr = None
    return _ocr


def read_shirt_number(crop: np.ndarray, ocr) -> list[str]:
    if ocr is None or crop is None or crop.size == 0:
        return []
    try:
        results = ocr.ocr(crop)
    except Exception:
        logger.exception("OCR inference failed")
        return []
    if not results:
        return []
    numbers = []
    for detection in results:
        try:
            _, (text, confidence) = detection
        except (ValueError, TypeError):
            continue
        if confidence > 0.5 and text.isdigit() and len(text) <= 2:
            numbers.append(text)
    return numbers


def verify_player(crop: np.ndarray, expected_number: int, ocr) -> bool:
    if ocr is None:
        return False
    numbers = read_shirt_number(crop, ocr)
    if not numbers:
        return False
    expected_str = str(expected_number)
    for detected in numbers:
        if detected == expected_str:
            return True
        if len(expected_str) > 1 and detected == expected_str[-1]:
            return True
    return False


def find_best_player_by_shirt(
    detections: list[dict], frame: np.ndarray, expected_number: int
) -> Optional[dict]:
    if not detections:
        return None

    ocr_instance = get_ocr()
    frame_cx = frame.shape[1] / 2.0
    frame_cy = frame.shape[0] / 2.0

    if ocr_instance is None:
        return min(
            detections,
            key=lambda d: np.hypot(
                d["center"][0] - frame_cx, d["center"][1] - frame_cy
            ),
        )

    sorted_dets = sorted(
        detections,
        key=lambda d: np.hypot(
            d["center"][0] - frame_cx, d["center"][1] - frame_cy
        ),
    )

    for detection in sorted_dets:
        crop = crop_player(frame, detection["bbox"])
        if verify_player(crop, expected_number, ocr_instance):
            return detection

    return sorted_dets[0]
