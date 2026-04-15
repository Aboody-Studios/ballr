# AI & Video Pipeline Specification: Ballr

## 1. Pipeline Overview
The pipeline transforms raw `90-minute` match footage into structured player insights and conversational coaching advice.

[Video] -> [Preprocessing] -> [Detection & Tracking] -> [Pose Estimation] -> [Event Extraction] -> [LLM Reasoning]

## 2. Computer Vision (CV) Modules

### 2.1 Preprocessing & Field Calibration
- **Object:** Extract 2D pitch coordinates from 2D video frames.
- **Tech:** OpenCV, Homography estimation (H-matrix).
- **Goal:** Transform pixel $(x, y)$ to pitch meters $(X, Y)$.

### 2.2 Player Detection & Tracking
- **Model:** YOLOv26 (Trained on football datasets).
- **Re-Identification (Re-ID):** StrongSORT or ByteTrack (Handling occlusions/player overlap).
- **Shirt Number Recognition:** Custom OCR (PaddleOCR or Tesseract) combined with Re-ID to link shirt numbers to tracking IDs.

### 2.3 Pose Estimation & Biomechanics
- **Model:** MediaPipe BlazePose or HRNet.
- **Metrics:** Scanning frequency, body orientation, acceleration, deceleration.

## 3. Data Extraction & Logic

### 3.1 Event Extraction
- **Passing:** Detect ball trajectory and proximity to players.
- **Positioning:** Spatial analysis (Heatmaps, distance from teammates/opponents).
- **Movement:** Total distance, sprint speed, intensity zones.

### 3.2 Positional Analytics
- **Standard Pitch Grid:** 105x68m mapping.
- **Output:** Time-series of $(X, Y)$ coordinates for the targeted player and the ball.

## 4. AI Coach Reasoning (LLM)

### 4.1 Input Context (Prompt Engineering)
The LLM (Gemini/GPT-4) receives:
1. **User Profile:** "18yo CM, wants to improve progressive passing."
2. **Global Stats:** "Pass accuracy 75%, 2 key passes, 5km covered."
3. **Event Logs (JSON):**
   - `{ "time": "15:20", "event": "PASS", "result": "INCOMPLETE", "context": "In defensive third, under pressure" }`
   - `{ "time": "42:10", "event": "MOVEMENT", "issue": "STATIC", "context": "Teammate in possession, no supporting run" }`

### 4.2 Reasoning Strategy
- **Chain-of-Thought:** Analyze *why* a mistake occurred (e.g., "Scanning was zero before receiving under pressure").
- **Persona:** Professional, encouraging, data-driven football coach.

## 5. Technical Stack
- **Language:** Python (for CV workers) & Go (for orchestration).
- **Inference Hardware:** NVIDIA GPUs (CUDA) in cloud or self-hosted.
- **Model Frameworks:** PyTorch, ONNX, TensorRT (for optimized inference).

## 6. Challenges & Mitigations
- **Occlusions:** Use Kalman filtering and Re-ID to maintain player ID.
- **Wide Angles:** Use high-resolution input (1080p/4K) for detecting shirt numbers.
- **Cost:** Batch processing and frame skipping (e.g., analyze 5-10 FPS instead of 30/60).
