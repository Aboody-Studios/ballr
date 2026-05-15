# AI & Video Pipeline Specification: Ballr

## 1. Scope: Single-Player Performance Analysis

Ballr analyzes **one player at a time** from match footage. The output is a structured biomechanical and tactical profile used by the AI Coach. Multi-person tracking and re-identification are out of scope -- the pipeline focuses entirely on the target player's movements, ball interactions, and event execution.

## 2. Pipeline

```
[Video] -> [Preprocessing] -> [Player Detection] -> [Pose Estimation] -> [Ball Detection] -> [Event Extraction] -> [LLM Reasoning]
```

## 3. YOLOv26 Model Classification (by task)

Ultralytics YOLOv26 (September 2025) is the foundation. It introduces NMS-free end-to-end inference, Residual Log-Likelihood Estimation (RLE) for pose, MuSGD optimizer, and 43% faster CPU inference over prior generations.

### 3.1 Primary: YOLOv26-pose (Pose Estimation)

**Purpose:** Full-body 17-keypoint detection for biomechanical analysis.

| Model | Params (M) | FLOPs (B) | mAP50-95 | mAP50 | CPU ONNX (ms) | T4 TensorRT (ms) | Use Case |
|-------|-----------|-----------|----------|-------|---------------|-------------------|----------|
| yolo26n-pose | 2.9 | 7.5 | 57.2 | 83.3 | 40.3 | 1.8 | Real-time edge/mobile inference |
| yolo26s-pose | 10.4 | 23.9 | 63.0 | 86.6 | 85.3 | 2.7 | Balanced speed/accuracy |
| yolo26m-pose | 21.5 | 73.1 | 68.8 | 89.6 | 218.0 | 5.0 | Offline high-accuracy analysis |
| yolo26l-pose | 25.9 | 91.3 | 70.4 | 90.5 | 275.4 | 6.5 | Server-side batch processing |
| yolo26x-pose | 57.6 | 201.7 | 71.6 | 91.6 | 565.4 | 12.2 | Maximum accuracy (research) |

**17 COCO keypoints:** nose, eyes, ears, shoulders, elbows, wrists, hips, knees, ankles.

**Architecture innovations relevant to Ballr:**
- **RLE (Residual Log-Likelihood Estimation):** Learns keypoint uncertainty from data. Produces more accurate joint localization on unusual poses (sprinting, sliding, mid-air headers) compared to fixed-distribution priors.
- **NMS-free inference:** Removes Non-Maximum Suppression post-processing. Latency is predictable and deterministic -- critical for frame-synchronous analysis.
- **MuSGD optimizer:** Hybrid SGD + Muon algorithm. More stable training on sports-specific datasets.

**Training recommendation:** Fine-tune `yolo26m-pose.pt` on a football/soccer-specific pose dataset (e.g., SoccerNet pose, or custom-annotated match footage) for the target player's sport.

### 3.2 Secondary: YOLOv26 (Object Detection)

**Purpose:** Ball detection relative to the tracked player.

| Model | Params (M) | mAP50-95 | mAP50 | T4 TensorRT (ms) | Use Case |
|-------|-----------|----------|-------|-------------------|----------|
| yolo26n | 2.9 | 44.0 | 62.3 | 1.5 | Real-time edge ball detection |
| yolo26s | 10.4 | 50.0 | 68.6 | 2.4 | Balanced |
| yolo26m | 21.5 | 54.7 | 73.2 | 4.7 | Server-side batch |

**Key improvement for Ballr:** YOLOv26 introduces ProgLoss + STAL (Spatial-Temporal Adaptive Loss) which significantly improves **small-object detection accuracy**. This is critical for ball detection (which can be <20px in wide-angle footage).

**Training recommendation:** Fine-tune `yolo26m.pt` on a custom ball dataset with motion-blur augmentation to handle high-speed ball movement (shots >100km/h).

### 3.3 YOLOE-26 (Open-Vocabulary Detection)

**Purpose:** Text-promptable detection for ad-hoc queries ("find the ball", "find the referee").

Reuses YOLOv26's end-to-end architecture. Not needed for core pipeline, but useful for interactive analysis features.

### 3.4 Not Used

- **YOLOv26-seg (Segmentation):** Instance segmentation is unnecessary for single-player analysis. Bounding box + keypoints provide sufficient spatial information.
- **YOLOv26-obb (Oriented BB):** Oriented bounding boxes add complexity without benefit for person/ball detection.
- **Multi-person Re-ID (ByteTrack/StrongSORT):** Since the pipeline tracks a single identified player, re-identification across occlusions is handled by spatial priors and temporal interpolation, not by multi-target tracking.

## 4. Computer Vision (CV) Modules

### 4.1 Preprocessing & Field Calibration
- **Object:** Extract 2D pitch coordinates from 2D video frames.
- **Tech:** OpenCV, Homography estimation (H-matrix).
- **Goal:** Transform pixel $(x, y)$ to pitch meters $(X, Y)$.

### 4.2 Player Detection
- **Model:** YOLOv26-pose (pretrained on COCO, fine-tuned on football datasets).
- **Input:** Full video frame (1080p recommended, subsampled to 640x640 for model).
- **Output:** Bounding box + 17 keypoints for the target player.
- **Single-player optimization:** Use the known shirt number from user input to filter detections. If the target player's jersey number is provided, run OCR on detected player crops to confirm identity. PaddleOCR (lightweight, ~5ms per crop).

### 4.3 Pose Estimation & Biomechanics
- **Model:** YOLOv26-pose (fine-tuned).
- **Extracted metrics per frame:**
  - Joint angles (hip, knee, ankle, shoulder, elbow) in degrees
  - Body orientation (direction of torso relative to goal)
  - Center of mass velocity and acceleration
  - Stride length and frequency (from hip-ankle distance deltas)
  - Head orientation (from nose-ear line, indicates scanning behavior)
- **Temporal smoothing:** 1D Kalman filter per keypoint coordinate to reduce jitter.

### 4.4 Ball Detection
- **Model:** YOLOv26-detect (fine-tuned on ball-only).
- **Output:** Bounding box center + confidence per frame.
- **Relation to player:** Compute distance-from-player, ball velocity vector, angle of approach/release.

### 4.5 Event Extraction

Events are classified from the pose + ball time-series, not from raw video:

| Event Type | Trigger Condition | Output Fields |
|-----------|-------------------|---------------|
| PASS | Ball velocity changes direction near player foot keypoints | timestamp, start_pos, end_pos, result (complete/incomplete) |
| SHOT | Ball velocity > threshold near player + arm swing pattern | timestamp, target (goal/wide/saved), speed |
| DRIBBLE | Ball within 2m of player feet for >3 frames + player moving | timestamp, start_pos, end_pos, touches |
| SPRINT | Player speed > 7m/s for >1 second | timestamp, distance, max_speed |
| TACKLE | Sudden deceleration + body orientation change + ball proximity | timestamp, result (won/lost) |
| RECOVERY | Player moves toward own goal at sprint speed without ball | timestamp, start_pos, end_pos |
| SCANNING | Head keypoint orientation change >30deg without body direction change | timestamp, count |

## 5. Positional Analytics

- **Standard Pitch Grid:** 105x68m mapping via homography.
- **Output:** Time-series of $(X, Y)$ coordinates for the target player and the ball.
- **Heatmaps:** Position density over the match period, zoned by pitch thirds.

## 6. AI Coach Reasoning (LLM)

### 6.1 Input Context (Prompt Engineering)
The LLM (Gemini 3.0 flash / DeepSeek v4 flash) receives:

1. **User Profile:** "18yo CM, wants to improve progressive passing."
2. **Global Stats:** "Pass accuracy 75%, 2 key passes, 5km covered."
3. **Event Logs (JSON):**
   ```json
   { "time": "15:20", "event": "PASS", "result": "INCOMPLETE", "context": "In defensive third, under pressure" }
   { "time": "42:10", "event": "MOVEMENT", "issue": "STATIC", "context": "Teammate in possession, no supporting run" }
   ```
4. **Biomechanical metrics:** "Avg stride length 1.2m, max sprint 8.3m/s, head scanning rate 0.3Hz (below threshold)."

### 6.2 Reasoning Strategy
- **Chain-of-Thought:** Analyze *why* a mistake occurred (e.g., "Scanning was zero before receiving under pressure").
- **Persona:** Professional, encouraging, data-driven football coach.

## 7. Technical Stack

| Component | Technology | Justification |
|-----------|-----------|---------------|
| CV Worker Language | Python | PyTorch/Ultralytics ecosystem, ONNX Runtime |
| Orchestration | Go | Existing backend (Redis Streams for job queue) |
| Model Framework | PyTorch (train) / ONNX (deploy) | ONNX for portable inference, TensorRT for GPU |
| Inference Hardware | NVIDIA T4 or better | Cloud GPU |
| Frame Sampling | 5-10 FPS | Reduces cost 3-6x vs 30fps, sufficient for pose |
| Input Resolution | 640x640 (model) / 1080p (raw) | YOLOv26 native. 1080p raw for OCR crops |

## 8. Deployment Architecture

```
[Raw Video (S3)] -> [Frame Sampler (Go)] -> [Python CV Worker]
                                                 |
                    +----------------------------+----------------------------+
                    |                            |                            |
            YOLOv26-pose                   YOLOv26-detect            PaddleOCR
            (17 keypoints)                (ball bbox)                (shirt number)
                    |                            |                            |
                    +----------------------------+----------------------------+
                                                 |
                                      [Event Extractor (Python)]
                                      [Kalman Smoothing (Python)]
                                                 |
                                      [Structured JSON -> Redis Streams]
                                                 |
                                      [AI Coach (LLM) -> Go API and Insights on the Dashboard]
```

## 9. Pre-Trained Models on Hugging Face

No custom training is required to start. All models below work out-of-box.

### 9.1 Player + Ball + Referee Detection

| Rank | Model | Base Arch | Classes | Performance | Source | License |
|------|-------|-----------|---------|-------------|--------|---------|
| 1 | **Roboflow Football Players** | YOLOv8/v9/v11 + RF-DETR | Player, GK, Referee, Ball | Most production-ready. Multiple archs. | [roboflow universe](https://universe.roboflow.com/roboflow-jvuqo/football-players-detection-3zvbc) | Open |
| 2 | **Soccana** | YOLOv11n + SAHI | Player, Ball, Referee | mAP 0.91, 30+ FPS at 1280x1280. Full analysis pipeline. | [hf.co/Adit-jain/soccana](https://huggingface.co/Adit-jain/soccana) | Open |
| 3 | `mobadam/football-player-detection` | YOLOv26l | Ball, Player, Referee, GK | Latest YOLO26, better small-object detection | [hf.co/mobadam](https://huggingface.co/mobadam/football-player-detection) | Apache-2.0 |
| 4 | `uisikdag/yolo-v8-football-players-detection` | YOLOv8 | Ball, GK, Player, Referee | mAP@0.5 = 0.785. Has working demo space. | [hf.co/uisikdag](https://huggingface.co/uisikdag/yolo-v8-football-players-detection) | Open |
| 5 | `martinjolif/yolo-football-player-detection` | YOLO11m | Player, GK, Referee, Ball | Player mAP50: 0.994, Ball: 0.680 | [hf.co/martinjolif](https://huggingface.co/martinjolif/yolo-football-player-detection) | AGPL-3.0 |
| 6 | **SportsVision-YOLO** | YOLOv8 | Player, Ball, Logo | 1300 frames from Eliteserien/Allsvenskan. Edge cases (airborne ball, foot-adhered). | [github](https://github.com/forzasys-students/SportsVision-YOLO) | Open |
| 7 | `Davidsv/football_insight` | YOLOv26m | Player, Ball, Referee | Team classification, formation lines, possession stats | [hf.co/Davidsv](https://huggingface.co/Davidsv/football_insight) | MIT |

**Recommendation for Ballr:** 'Roboflow Football Players' offers the widest architecture choice (YOLOv8/v9/v11 + RF-DETR) and is the most production-ready. 'Soccana' is the best pick for a single integrated pipeline on HuggingFace. Use 'mobadam/football-player-detection' if you specifically want the latest YOLOv26 architecture.

### 9.2 Ball Detection (dedicated, for hard cases)

| HF Model | Base | mAP50 | Notes |
|----------|------|-------|-------|
| `martinjolif/yolo-football-ball-detection` | YOLO11n | 89.1% | Dedicated ball-only detector. Small object specialist. 1.24k downloads. |

**Recommendation:** Use as a second-stage detector if the primary model's ball class is insufficient. The `Soccana` model handles ball detection at mAP 0.91 within its multi-class setup. For higher-resolution ball detection, YOLOv5m at 1280px achieves ~60% better mAP50-95 than 640px (per `Football-Tracking` benchmarks).

### 9.3 Pose Estimation (biomechanics)

| HF Model | Base | mAP50-95 | T4 Latency | Params | Use |
|----------|------|----------|------------|--------|-----|
| `openvision/yolo26n-pose` | YOLOv26n | 57.2 | 1.8ms | 2.9M | **Edge / real-time** |
| `openvision/yolo26s-pose` | YOLOv26s | 63.0 | 2.7ms | 10.4M | Balanced |
| `openvision/yolo26m-pose` | YOLOv26m | **68.8** | **5.0ms** | 21.5M | **Cloud: best accuracy/cost** |
| `openvision/yolo26l-pose` | YOLOv26l | 70.4 | 6.5ms | 25.9M | Server-side batch |
| `openvision/yolo26x-pose` | YOLOv26x | 71.6 | 12.2ms | 57.6M | Max accuracy |
| `onnx-community/yolo26m-pose-ONNX` | YOLOv26m | 68.8 | - | 21.5M | ONNX Runtime, no PyTorch |
| `onnx-community/yolo26x-pose-ONNX` | YOLOv26x | 71.6 | - | 57.6M | ONNX, max accuracy |

17 COCO keypoints: nose, eyes, ears, shoulders, elbows, wrists, hips, knees, ankles.

**Recommendation:** `openvision/yolo26m-pose` for cloud. `onnx-community/yolo26m-pose-ONNX` for ONNX Runtime deployment.

### 9.4 Pitch Keypoint Detection (homography / tactical view)

| Model | Base Arch | Keypoints | Performance | Source |
|-------|-----------|-----------|-------------|--------|
| `Simon9/football-field-detection-roboflow` | YOLOv8 keypoint | Field lines + corners | Trained on Roboflow dataset | [hf.co/Simon9](https://huggingface.co/Simon9/football-field-detection-roboflow) |
| `martinjolif/yolo-football-pitch-detection` | YOLOv8x-pose | 32 | Field lines, corners, markings. 317 downloads. | [hf.co/martinjolif](https://huggingface.co/martinjolif/yolo-football-pitch-detection) |
| `Adit-jain/Soccana_Keypoint` | YOLOv11-pose | 29 | 94.2% detection rate. SoccerNet-derived. | Part of [Soccer_Analysis](https://github.com/Adit-jain/Soccer_Analysis) |

**Recommendation:** Use for converting pixel coordinates to pitch meters via homography.

### 9.5 Edge / Quantized Models

| HF Model | Base | Quant | Size | Target |
|----------|------|-------|------|--------|
| `mlx-community/YOLO26n-OptiQ-6bit` | YOLO26n | 6-bit | ~3MB | Apple Silicon |
| `mlx-community/YOLO26s-OptiQ-6bit` | YOLO26s | 6-bit | ~6MB | Apple Silicon |
| `mlx-community/YOLO26m-OptiQ-6bit` | YOLO26m | 6-bit | ~12MB | Apple Silicon |
| `mlx-community/YOLO26l-OptiQ-6bit` | YOLO26l | 6-bit | ~15MB | Apple Silicon |
| `mlx-community/YOLO26x-OptiQ-6bit` | YOLO26x | 6-bit | ~35MB | Apple Silicon |

### 9.6 Complete Open-Source Pipelines (Tracking + Tactical)

These provide more than a model -- full tracking, team classification, tactical analysis.

| Project | Models Used | Features | Source |
|---------|-----------|----------|--------|
| [Adit-jain/Soccer_Analysis](https://github.com/Adit-jain/Soccer_Analysis) | YOLOv11 + ByteTrack + SigLIP | Detection, tracking, team assignment (SigLIP/UMAP/K-Means), 29-keypoint pitch, homography | GitHub |
| [Darkmyter/Football-Players-Tracking](https://github.com/Darkmyter/Football-Players-Tracking) | YOLOv8 + ByteTrack | Multi-object tracking. YOLOv5m at 1280px = 60% better ball mAP50-95 vs 640px. | GitHub |
| [WWandP/SoccerEye](https://github.com/WWandP/SoccerEye) | YOLOv8x_1280 + MaskRCNN | Bird's-eye view, player grouping, speed, trajectory. 5 scales at 640/1280. | GitHub |
| [francescopiocirillo/soccer-players-tracking](https://github.com/francescopiocirillo/soccer-players-tracking) | YOLO12m + Bot SORT + OSNet | SoccerNet 2023, camera motion compensation, HOTA 0.762 | GitHub |
| [Smasko7/Football-Vision](https://github.com/Smasko7/Football-Vision) | Roboflow YOLO + ByteTrack | Unsupervised team classification, Voronoi, 2D pitch radar | GitHub |
| [rustyneuron01/Score-Vision](https://github.com/rustyneuron01/Real-Time-Football-Detection) | YOLO + HRNet + OSNet + ByteTrack | Distributed inference, FastAPI, CLIP/VLM validation | GitHub |

**Note:** These pipelines are full-match multi-person systems. Ballr's single-player focus means we simplify significantly: no Re-ID, no team classification, no multi-target tracking. The resolution insight from `Football-Players-Tracking` (1280px = 60% better ball detection) is directly applicable.

### 9.7 Related Tasks (Out of Scope but Noted)

| Task | Project | Approach | Source |
|------|---------|----------|--------|
| Action Spotting | SoccerNet / T-DEED | Temporal model, 12 action classes (Pass, Drive, Header, Shot, etc.) | [github.com/SoccerNet](https://github.com/SoccerNet) |
| Offside Detection | SoccerOffsideTracker | YOLOv8 seg + jersey color clustering + perspective transform | [github](https://github.com/AggieSportsAnalytics/SoccerOffsideTracker) |
| Player Depth | SoccerNet MDE 2025 | YOLOv8 seg + DepthAnything + UNet correction (35-57% better) | arXiv |

These are not needed for initial Ballr pipeline but may become relevant for future features (automatic event labeling, offside analysis, 3D reconstruction).

### 9.8 Zero-Training Stack (recommended to start)

```python
from ultralytics import YOLO

# Option A: Soccana (HF, single model for detection)
detector = YOLO("Adit-jain/soccana")  # YOLOv11n, mAP 0.91

# Option B: Roboflow (download .pt from universe.roboflow.com)
# Supports YOLOv8/v9/v11 + RF-DETR architectures
detector = YOLO("football-players-detection.pt")

# Pose estimation for biomechanics (17 COCO keypoints)
pose = YOLO("openvision/yolo26m-pose")  # 68.8 mAP50-95, 5ms on T4

# Pitch calibration for pixel->meter coordinate mapping
pitch = YOLO("martinjolif/yolo-football-pitch-detection")  # 32 keypoints

# Inference
players = detector(frame)      # -> player/ball bbox + class
skeleton = pose(frame)         # -> 17 keypoints per person
field = pitch(frame)           # -> pitch landmarks for homography
```

No training data or fine-tuning required.

### 9.9 Quick Reference Table (all tasks)

| Task | Recommended Model | YOLO Version | Source |
|------|------------------|-------------|--------|
| Player/Ball/Ref Detection | Roboflow Football Players + Soccana | v8/v9/v11/YOLOv26 + RF-DETR | Roboflow Universe + HuggingFace |
| Ball Detection (hard cases) | `martinjolif/yolo-football-ball-detection` | YOLO11n | HuggingFace |
| Pose Estimation | `openvision/yolo26m-pose` | YOLOv26m | HuggingFace |
| Multi-Object Tracking | Football-Tracking (ByteTrack) | YOLOv8 + ByteTrack | GitHub |
| Field Keypoints / Homography | `Simon9` / `martinjolif` / `Soccana_Keypoint` | YOLOv8/11 keypoint | HuggingFace |
| Action Spotting (temporal) | SoccerNet / T-DEED | - | GitHub |
| Offside Detection | SoccerOffsideTracker | YOLOv8 seg | GitHub |
| Player Depth | SoccerNet MDE 2025 | YOLOv8 seg + DepthAnything | arXiv |

## 10. Ball-Specific Challenges

Ball detection is the hardest task in football CV due to small size (~20px), motion blur (100+ km/h), and occlusion. Mitigation strategies:

| Strategy | When | How |
|----------|------|-----|
| Dedicated ball detector | Ball accuracy <80% | `martinjolif/yolo-football-ball-detection` (89% mAP50) |
| Temporal interpolation | Ball disappears for <1s | Kalman filter on ball trajectory |
| Frame interpolation | High-speed shots (blur) | TrackNet-style 3-frame temporal window |
| High-res inference | Ball <20px | Run detector at 1280px (SAHI tiling if needed) |

For single-player focus, ball tracking is simpler: we know the target player's position, so we search for the ball in a region of interest near the player.

## 11. Training Data Requirements (if fine-tuning)

| Model | Base Model | Fine-tune Data | Samples Needed |
|-------|-----------|---------------|---------------|
| YOLOv26-pose | `openvision/yolo26m-pose` | Football-specific keypoint annotations (17-keypoint) | 2,000-5,000 frames |
| YOLOv26-detect (ball) | `Adit-jain/soccana` | Ball-only with motion blur augmentation | 1,000-3,000 frames |
| Event classifier | - | Pose sequence + ball trajectory labeled with event types | 5,000-10,000 sequences |

## 12. Challenges & Mitigations

| Challenge | Mitigation |
|-----------|-----------|
| Occlusions (player hidden behind others) | Kalman filter interpolation + single-player focus (know target's approximate position) |
| Ball motion blur at high speed | Frame sampling at 1/1000s shutter equivalent or temporal ball detection (TrackNet-style on 3-frame windows) |
| Illumination changes across stadiums | YOLOv26's ProgLoss + STAL; data augmentation during fine-tuning (brightness, contrast, noise) |
| Processing cost | Frame skipping (5 FPS); batch inference; ONNX Runtime with TensorRT EP |
| Pose drift over long matches | Extended Kalman filter with biomechanical constraints (joint angle limits) |

## 13. Model Selection Guide (with HF Sources)

| Deployment | Ballr Use Case | Detection Model | Pose Model | Detection HF Source | Pose HF Source |
|-----------|---------------|----------------|------------|---------------------|----------------|
| Cloud GPU (T4+) | Offline analysis | YOLOv11n | yolo26m-pose | `Adit-jain/soccana` | `openvision/yolo26m-pose` |
| Cloud GPU (T4+) | Offline (latest arch) | YOLOv26l | yolo26m-pose | `mobadam/football-player-detection` | `openvision/yolo26m-pose` |
| Cloud GPU (A100) | Batch processing | YOLOv26l | yolo26l-pose | `mobadam/football-player-detection` | `openvision/yolo26l-pose` |
| Edge (Jetson) | Real-time feedback | YOLOv11n | yolo26n-pose (ONNX) | `Adit-jain/soccana` | `onnx-community/yolo26n-pose-ONNX` |
| Apple Silicon | Live form check | YOLO26m (6-bit) | yolo26m-pose (6-bit) | `mlx-community/YOLO26m-OptiQ-6bit` | `onnx-community/yolo26m-pose-ONNX` |
