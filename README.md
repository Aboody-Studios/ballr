# Ballr: System Overview

Ballr is a distributed platform designed to provide amateur football players with professional-grade performance analytics and AI-driven coaching. The system ingests match footage, processes it through a Computer Vision (CV) pipeline to extract spatial and event-based metrics, and uses Large Language Models (LLMs) to provide personalized feedback and training plans.

## System Purpose

The core mission of Ballr is to bridge the gap between video recording and actionable athletic improvement. It automates the extraction of metrics such as top speed, pass accuracy, and heatmaps from standard video files, then contextualizes this data within a gamified progress system.

## High-Level Architecture

Ballr utilizes a multi-domain hexagonal architecture implemented in Go for the backend, an asynchronous Python-based Computer Vision pipeline, and a Flutter mobile client. The backend is organized into four primary domains: **Identity**, **Match**, **Coach**, and **Progress**.

## Key Domains

The system is partitioned into bounded contexts to ensure maintainability and clear separation of concerns.

|Domain|Responsibility|Key Code Entities|
|---|---|---|
|**Identity**|Authentication and User Profiles|`IdentityHandler`, `IdentityService`, `PostgresUserRepo`|
|**Match**|Video orchestration and CV analysis|`UploadHandler`, `AnalysisService`, `AnalysisWorker`|
|**Coach**|AI reasoning and training plans|`CoachHandler`, `CoachService`, `LLMProvider`|
|**Progress**|Gamification, XP, and Streaks|`ProgressHandler`, `GamificationService`, `EventLog`|

## Technology Stack

The project leverages modern frameworks and managed services to handle high-compute video processing and low-latency API responses.

- **Backend:** Go 1.25+ using the Echo framework.
- **Database:** PostgreSQL for relational data managed via GORM.
- **Asynchronous Messaging:** Redis Streams for event-driven domain communication.
- **Computer Vision:** Python 3.12-based pipeline using YOLO and Pose Estimation.
- **Storage:** AWS S3 for video files and generated assets.
- **AI/LLM:** OpenAI and Google Gemini for coaching logic.

## Data Flow: Video to Insight

The transition from "Natural Language Space" (a user uploading a video) to "Code Entity Space" (data persistence) follows a strict pipeline:

1. **Ingestion:** `UploadService` generates a presigned URL via `StorageRepository`.
2. **Trigger:** Client notifies API; `AnalysisService` pushes a job to `RedisJobQueue`.
3. **Processing:** `AnalysisWorker` (Go) invokes the CV script defined in `CV_SCRIPT_PATH` as a subprocess.
4. **Analysis:** Python pipeline (src/cv) extracts player metrics and generates heatmaps.
5. **Persistence:** Results are stored via `PostgresAnalysisRepository` and `PostgresMatchRepository`.
6. **Gamification:** Events like `EventAnalysisCompleted` are published to `Redis Streams`, which `GamificationService` consumes to `GrantPoints`.
