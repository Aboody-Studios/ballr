import json
import logging
import os
from typing import Optional

import numpy as np

import config

logger = logging.getLogger(__name__)

_AGG_SET = False


def _ensure_agg():
    global _AGG_SET
    if not _AGG_SET:
        import matplotlib
        matplotlib.use("Agg")
        _AGG_SET = True


def generate_heatmap(positions: list[tuple[float, float]], output_path: str, title: str) -> str:
    _ensure_agg()
    import matplotlib.pyplot as plt
    from matplotlib.colors import LinearSegmentedColormap
    from scipy.ndimage import gaussian_filter

    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    if not positions:
        fig, ax = plt.subplots(1, 1, figsize=(10.5, 6.8))
        ax.set_xlim(0, config.CV_HEATMAP_WIDTH)
        ax.set_ylim(0, config.CV_HEATMAP_HEIGHT)
        ax.set_title(title)
        ax.set_aspect("equal")
        fig.savefig(output_path, dpi=100, bbox_inches="tight")
        plt.close(fig)
        return output_path

    xs = np.array([p[0] for p in positions])
    ys = np.array([p[1] for p in positions])
    xbins = config.CV_HEATMAP_WIDTH
    ybins = config.CV_HEATMAP_HEIGHT
    heatmap, _, _ = np.histogram2d(ys, xs, bins=[ybins, xbins], range=[[0, config.CV_HEATMAP_HEIGHT], [0, config.CV_HEATMAP_WIDTH]])
    heatmap = gaussian_filter(heatmap.astype(float), sigma=8.0)
    heatmap = heatmap / (heatmap.max() + 1e-8)
    cmap = LinearSegmentedColormap.from_list("pitch_heat", ["#000033", "#0033cc", "#00ff00", "#ffff00", "#ff6600", "#ff0000"], N=256)
    fig, ax = plt.subplots(1, 1, figsize=(10.5, 6.8))
    ax.imshow(heatmap, cmap=cmap, origin="lower", extent=[0, config.CV_HEATMAP_WIDTH, 0, config.CV_HEATMAP_HEIGHT], aspect="auto")
    ax.set_xlim(0, config.CV_HEATMAP_WIDTH)
    ax.set_ylim(0, config.CV_HEATMAP_HEIGHT)
    ax.set_title(title, fontsize=14, pad=10)

    mid_x = config.CV_HEATMAP_WIDTH // 2
    ax.axvline(x=mid_x, color="white", linestyle="-", linewidth=0.5, alpha=0.3)
    ax.plot([0, config.CV_HEATMAP_WIDTH, config.CV_HEATMAP_WIDTH, 0, 0],
            [0, 0, config.CV_HEATMAP_HEIGHT, config.CV_HEATMAP_HEIGHT, 0],
            color="white", linewidth=0.8, alpha=0.4)
    ax.axis("off")
    fig.savefig(output_path, dpi=100, bbox_inches="tight", pad_inches=0, transparent=False, facecolor="#1a1a2e")
    plt.close(fig)
    return output_path


def _upload_to_s3(file_path: str, bucket: str, s3_key: str) -> str:
    import boto3
    s3 = boto3.client("s3", region_name=config.CV_S3_REGION)
    try:
        s3.upload_file(file_path, bucket, s3_key, ExtraArgs={"ContentType": "image/png"})
        region = config.CV_S3_REGION
        url = f"https://{bucket}.s3.{region}.amazonaws.com/{s3_key}"
        logger.info("uploaded %s to s3://%s/%s", file_path, bucket, s3_key)
        return url
    except Exception as e:
        logger.error("s3 upload failed for %s: %s", file_path, e)
        return ""


def _scale_pos(
    pc: tuple[float, float], H: Optional[np.ndarray] = None
) -> tuple[float, float]:
    if H is not None:
        try:
            from pitch import pixel_to_pitch

            x_pct, y_pct = pixel_to_pitch(pc[0], pc[1], H)
            sx = x_pct * (config.CV_HEATMAP_WIDTH / 100.0)
            sy = y_pct * (config.CV_HEATMAP_HEIGHT / 100.0)
            return (
                max(0, min(config.CV_HEATMAP_WIDTH, sx)),
                max(0, min(config.CV_HEATMAP_HEIGHT, sy)),
            )
        except Exception:
            pass
    sx = pc[0] * (config.CV_HEATMAP_WIDTH / 1920.0)
    sy = pc[1] * (config.CV_HEATMAP_HEIGHT / 1080.0)
    return (max(0, min(config.CV_HEATMAP_WIDTH, sx)),
            max(0, min(config.CV_HEATMAP_HEIGHT, sy)))


def generate_all_heatmaps(
    player_history: list[dict],
    match_id: str,
    output_dir: str,
    bucket: str = "",
    homography: Optional[np.ndarray] = None,
) -> dict:
    os.makedirs(output_dir, exist_ok=True)
    all_positions = []
    defensive = []
    attacking = []
    for h in player_history:
        pc = h.get("player_center")
        if pc is None:
            continue
        scaled = _scale_pos(pc, homography)
        all_positions.append(scaled)
        if scaled[1] < config.CV_HEATMAP_HEIGHT / 2:
            defensive.append(scaled)
        else:
            attacking.append(scaled)

    files = {
        "overall": os.path.join(output_dir, f"{match_id}_overall.png"),
        "defensive": os.path.join(output_dir, f"{match_id}_defensive.png"),
        "attacking": os.path.join(output_dir, f"{match_id}_attacking.png"),
    }
    generate_heatmap(all_positions, files["overall"], f"Overall Positioning - {match_id}")
    generate_heatmap(defensive, files["defensive"], f"Defensive Half - {match_id}")
    generate_heatmap(attacking, files["attacking"], f"Attacking Half - {match_id}")

    urls: dict[str, str] = {
        "overall_url": "",
        "defensive_url": "",
        "attacking_url": "",
    }
    if bucket:
        for key, file_path in files.items():
            s3_key = f"heatmaps/{match_id}/{key}.png"
            url = _upload_to_s3(file_path, bucket, s3_key)
            urls[f"{key}_url"] = url
    return urls


def upload_tracking_data(history: list[dict], match_id: str, bucket: str = "") -> str:
    serializable = _make_serializable(history)
    if bucket:
        import json
        import tempfile
        tmp = tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False)
        json.dump(serializable, tmp, default=str)
        tmp.close()
        s3_key = f"tracking/{match_id}/data.json"
        url = _upload_to_s3(tmp.name, bucket, s3_key)
        os.unlink(tmp.name)
        return url
    return ""


def _make_serializable(history: list[dict]) -> list[dict]:
    result = []
    for h in history:
        entry = {}
        for k, v in h.items():
            if isinstance(v, np.ndarray):
                entry[k] = v.tolist()
            elif isinstance(v, (np.floating, np.integer)):
                entry[k] = v.item()
            elif isinstance(v, tuple):
                entry[k] = list(v)
            elif isinstance(v, dict):
                entry[k] = {sk: float(sv) if isinstance(sv, (int, float, np.floating, np.integer)) else sv
                            for sk, sv in v.items()}
            elif v is None or isinstance(v, (str, int, float, bool, list)):
                entry[k] = v
            else:
                try:
                    json.dumps(v)
                    entry[k] = v
                except (TypeError, OverflowError):
                    entry[k] = str(v)
        result.append(entry)
    return result
