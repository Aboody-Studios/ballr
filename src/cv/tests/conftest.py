import sys
from unittest.mock import MagicMock

import numpy as np
import pytest

_ultralytics_mock = MagicMock()
_ultralytics_mock.YOLO = MagicMock
sys.modules["ultralytics"] = _ultralytics_mock


class _IndexableMock:
    def __init__(self, items):
        self._items = items

    def __getitem__(self, i):
        m = MagicMock()
        item = self._items[i]
        if isinstance(item, list):
            m.tolist.return_value = item
        else:
            m.__float__ = lambda _=float(item): float(item)
            m.__int__ = lambda _=int(item): int(item)
            m.tolist.return_value = [item]
        return m

    def __len__(self):
        return len(self._items)

    def __bool__(self):
        return len(self._items) > 0


class _BoxesMock:
    def __init__(self, xyxy, conf, cls, names=None):
        self.xyxy = _IndexableMock(xyxy)
        self.conf = _IndexableMock(conf)
        self.cls = _IndexableMock(cls)
        self.names = names or {}
        self._box_len = len(xyxy)

    def __len__(self):
        return self._box_len


def _make_mock_keypoints(data_3d):
    mock = MagicMock()
    arr = np.array(data_3d, dtype=float)
    cpu_mock = MagicMock()
    cpu_mock.numpy.return_value = arr
    mock.keypoints = MagicMock()
    mock.keypoints.data = MagicMock()
    mock.keypoints.data.cpu.return_value = cpu_mock
    return mock


@pytest.fixture
def synthetic_frame():
    return np.random.randint(0, 256, (480, 640, 3), dtype=np.uint8)


@pytest.fixture
def mock_boxes():
    return _BoxesMock(
        [[100.0, 200.0, 300.0, 400.0],
         [350.0, 150.0, 500.0, 350.0],
         [50.0, 50.0, 120.0, 180.0]],
        [0.95, 0.85, 0.90],
        [0, 0, 0],
    )


@pytest.fixture
def mock_ball_boxes():
    return _BoxesMock(
        [[400.0, 300.0, 420.0, 320.0]],
        [0.8],
        [1],
    )


@pytest.fixture
def mock_detection_result(mock_boxes):
    result = MagicMock()
    result.boxes = mock_boxes
    result.names = {0: "player", 1: "ball"}
    return result


@pytest.fixture
def mock_ball_detection_result(mock_ball_boxes):
    result = MagicMock()
    result.boxes = mock_ball_boxes
    result.names = {0: "player", 1: "ball"}
    return result


@pytest.fixture
def mock_model(mock_detection_result):
    model = MagicMock()
    model.return_value = [mock_detection_result]
    return model


@pytest.fixture
def mock_pose_keypoints():
    kps = []
    for i in range(17):
        x = 200.0 + i * 5
        y = 300.0 + (i % 3) * 10
        conf = 0.95
        kps.append([x, y, conf])
    return kps


@pytest.fixture
def mock_pose_result(mock_pose_keypoints):
    keypoints_3d = [mock_pose_keypoints]
    return _make_mock_keypoints(keypoints_3d)


@pytest.fixture
def mock_pose_model(mock_pose_result):
    model = MagicMock()
    model.return_value = [mock_pose_result]
    return model


@pytest.fixture
def tracking_history():
    history = []
    for i in range(10):
        t = f"00:{i // 60:02d}:{i % 60:02d}"
        entry = {
            "frame_idx": i,
            "timestamp": t,
            "player_center": (320.0 + i * 2, 240.0 + i),
            "ball_center": (330.0 + i * 3, 250.0 + i + 1),
            "keypoints": [(200.0 + i, 300.0, 0.95) for _ in range(17)],
            "joint_angles": {"left_knee": 90.0 + i},
            "head_orientation": "center",
            "player_speed": 2.0 + i * 0.1,
            "player_acceleration": 0.1,
            "ball_speed": 1.0 + i * 0.2,
            "ball_velocity": (1.0, 0.5),
        }
        history.append(entry)
    return history


@pytest.fixture
def mock_s3_client():
    return MagicMock()
