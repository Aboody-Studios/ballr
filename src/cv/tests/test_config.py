import importlib

import config


def _reload_with_env(env_vars):
    for k, v in env_vars.items():
        import os
        if v is None:
            os.environ.pop(k, None)
        else:
            os.environ[k] = str(v)
    importlib.reload(config)


def test_cv_fps_target_default():
    _reload_with_env({"CV_FPS_TARGET": None})
    assert config.CV_FPS_TARGET == 10


def test_cv_fps_target_read_from_env():
    _reload_with_env({"CV_FPS_TARGET": "15"})
    assert config.CV_FPS_TARGET == 15


def test_cv_detection_conf_default():
    _reload_with_env({"CV_DETECTION_CONF": None})
    assert config.CV_DETECTION_CONF == 0.5


def test_cv_detection_conf_read_as_float():
    _reload_with_env({"CV_DETECTION_CONF": "0.75"})
    assert config.CV_DETECTION_CONF == 0.75
    assert isinstance(config.CV_DETECTION_CONF, float)


def test_cv_pose_conf_default():
    _reload_with_env({"CV_POSE_CONF": None})
    assert config.CV_POSE_CONF == 0.5


def test_cv_pose_conf_read_as_float():
    _reload_with_env({"CV_POSE_CONF": "0.6"})
    assert config.CV_POSE_CONF == 0.6
    assert isinstance(config.CV_POSE_CONF, float)


def test_cv_s3_bucket_default():
    _reload_with_env({"CV_S3_BUCKET": None})
    assert config.CV_S3_BUCKET == ""


def test_cv_s3_bucket_read_from_env():
    _reload_with_env({"CV_S3_BUCKET": "my-bucket"})
    assert config.CV_S3_BUCKET == "my-bucket"


def test_cv_heatmap_output_dir_default():
    _reload_with_env({"CV_HEATMAP_OUTPUT_DIR": None})
    assert config.CV_HEATMAP_OUTPUT_DIR == "/tmp/ballr-heatmaps"


def test_cv_px_per_meter_default():
    _reload_with_env({"CV_PX_PER_METER": None})
    assert config.CV_PX_PER_METER == 15.0


def test_cv_kalman_process_noise_default():
    _reload_with_env({"CV_KALMAN_PROCESS_NOISE": None})
    assert config.CV_KALMAN_PROCESS_NOISE == 1e-4


def test_cv_kalman_measurement_noise_default():
    _reload_with_env({"CV_KALMAN_MEASUREMENT_NOISE": None})
    assert config.CV_KALMAN_MEASUREMENT_NOISE == 1e-2


def test_cv_enable_ocr_default():
    _reload_with_env({"CV_ENABLE_OCR": None})
    assert config.CV_ENABLE_OCR is True


def test_cv_enable_ocr_disabled():
    _reload_with_env({"CV_ENABLE_OCR": "0"})
    assert config.CV_ENABLE_OCR is False
