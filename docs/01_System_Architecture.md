# System Architecture: Ballr

## 1. Overview
Ballr is a distributed system designed for processing high-volume video data, performing computer vision (CV) analysis, and providing an AI-driven coaching experience. The architecture follows a microservices-inspired approach with a Go-based backend and a Flutter mobile application.

## 2. High-Level Diagram
[Mobile App (Flutter)] <--> [API Gateway / Auth (Go)] <--> [Core Services (Go)]
                                                                    |
                                                                    v
[Video Storage (S3/GCS)] <--- [Analysis Worker (Go/Python)] <--- [Message Queue (Redis/RabbitMQ)]
                                        |
                                        v
[AI Engine (CV Models)] <--> [LLM Service (Coach reasoning)]

## 3. Key Components

### 3.1 Mobile Frontend (Flutter)
- **Architecture:** Clean Architecture with Riverpod for state management.
- **Responsibility:** User onboarding, video recording/uploading, interactive analysis reports, AI Coach chat interface, and gamification dashboard.

### 3.2 Backend API (Go)
- **Framework:** Fiber or Echo (Standard Library for core).
- **Responsibility:** User management, session handling, video metadata management, orchestration of analysis jobs, and serving AI Coach responses.

### 3.3 Analysis Pipeline
- **Ingestion:** Direct upload to Cloud Storage (S3/GCS) with pre-signed URLs.
- **Queueing:** Asynchronous job processing using Redis or RabbitMQ.
- **Processing Worker:** A hybrid service (Go for orchestration, Pythonfor CV models like YOLOv8/MediaPipe) that extracts player coordinates, ball trajectory, and match events.
- **Output:** Structured JSON data (positional data, event logs) stored in PostgreSQL/MongoDB.

### 3.4 AI Coach (LLM)
- **Engine:** Gemini Pro / Z.ai / MiMo.
- **Contextualization:** The LLM receives the structured analysis data (e.g., "Player moved too wide at 15:30") and generates natural language feedback.

### 3.5 Gamification & Progress
- **Logic:** Real-time event tracking in Go.
- **Persistence:** PostgreSQL for point tracking, streaks, and achievements.

## 4. Communication Protocols
- **Client-Server:** RESTful API for standard operations; WebSocket for real-time chat updates and upload status.
- **Internal:** gRPC or message queues for inter-service communication.
- **Video:** HLS/DASH for streaming analyzed clips to the mobile app.
