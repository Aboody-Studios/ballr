# Backend Service Specification: Ballr (Go)

## 1. Core Framework & Language
- **Language:** Go (Golang) 1.26+
- **Maintainer:** Omar Basem
- **Framework:** **Fiber** (High performance, Express-like) or **Echo**.
- **Package Management:** Go Modules.
- **Project Structure:** Clean Architecture (Standard Go Layout).
  - `/cmd`: Entry points (main.go).
  - `/internal`: Core logic, models, services, repositories.
  - `/pkg`: Shared utilities (auth, logger).
  - `/api`: OpenAPI/Swagger definitions.

## 2. Key Microservices (Logical Separation)

### 2.1 Identity Service
- **Authentication:** JWT (JSON Web Tokens) with refresh token rotation.
- **Provider:** Custom Auth or Firebase Auth integration for fast onboarding.
- **Endpoints:** `/auth/register`, `/auth/login`, `/auth/profile`.

### 2.2 Video & Analysis Orchestrator
- **Upload Strategy:** Pre-signed URLs for direct-to-S3 uploads.
- **Workflow:**
  1. Generate pre-signed URL for client.
  2. Client uploads video.
  3. Client notifies Backend (`POST /analysis/start`).
  4. Backend pushes job to Redis/SQS.
  5. Backend updates status via WebSockets or polling.
- **Endpoints:** `/analysis/upload-url`, `/analysis/status/:id`, `/analysis/report/:id`.

### 2.3 AI Coach Service
- **Interactions:** Conversational interface using an LLM (Gemini/OpenAI).
- **RAG (Retrieval-Augmented Generation):**
  - Prompt context includes: User profile (age, position), latest match analysis, and training history.
- **Endpoints:** `/coach/chat`, `/coach/plan/generate`, `/coach/diet/generate`.

### 2.4 Gamification Service
- **Logic:** Event-based point calculation (e.g., `MATCH_UPLOADED`, `DRILL_COMPLETED`).
- **Persistence:** PostgreSQL for ACID compliance on points/rewards.
- **Endpoints:** `/progress/summary`, `/achievements/list`, `/leaderboard`.

## 3. Database Strategy
- **Relational (PostgreSQL):** Users, match metadata, points, achievements, and drill content.
- **Document (MongoDB/PostgreSQL JSONB):** Complex match analysis results (positional data, event logs, coordinate time-series).
- **Cache (Redis):** Session management, rate limiting, and analysis job queues.

## 4. External Integrations
- **Cloud Storage:** AWS S3 or Google Cloud Storage (GCS).
- **Video Processing:** ffmpeg-based worker.
- **AI Models:** Vertex AI (Gemini) or OpenAI API or Z.ai or MiMo Providers.

## 5. Error Handling & Logging
- **Logging:** Structured logging with `zap` or `zerolog`.
- **Tracing:** OpenTelemetry for distributed tracing.
- **Monitoring:** Prometheus & Grafana.
