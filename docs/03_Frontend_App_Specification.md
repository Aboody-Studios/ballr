# Frontend App Specification: Ballr (Flutter)

## 1. Core Framework & Language
- **Language:** Dart 3.x+
- **Framework:** Flutter 3.x+ (Stable Channel)
- **Platforms:** iOS & Android (Mobile-first).

## 2. Architecture: Clean Architecture + Riverpod
- **Layers:**
  - **Presentation:** UI widgets, Riverpod providers, ViewModels.
  - **Domain:** Entities, Usecases, Repository interfaces.
  - **Data:** Repository implementations, Data sources (REST, Local DB).
- **State Management:** **Riverpod** (Functional & Reactive).
- **Navigation:** **go_router** (Declarative routing).

## 3. UI/UX Strategy
- **Design System:** Material 3 with custom brand colors (Modern, Sporty, High-Contrast).
- **Key Modules:**
  - **Auth:** Login/Register, Profile setup.
  - **Match Analysis:** Video picker, upload progress, interactive report viewer (charts, heatmaps).
  - **AI Coach:** Chat UI with bubbles, typing indicators, markdown support.
  - **Drill Library:** Searchable list, detail views with focus points.
  - **Dashboard:** Progress tracking, achievements, trophies.

## 4. Key Libraries
- **Video:** `video_player`, `chewie` (advanced controls).
- **Network:** `dio` (with interceptors for JWT).
- **Charts:** `fl_chart` (for performance heatmaps/graphs).
- **Animations:** `lottie` or `rive` (for gamification/achievement effects).
- **Local Storage:** `isar` (fast NoSQL) for offline drills/cache.
- **Assets:** SVG icons (`flutter_svg`).

## 5. Feature-Specific Logic

### 5.1 Match Upload Workflow
- **Chunked Upload:** For 90-minute videos (if direct S3 is too large/unstable).
- **Background Support:** `flutter_background_service` to continue uploads.
- **Progress Tracking:** Real-time feedback using Riverpod `AsyncValue`.

### 5.2 AI Coach Interface
- **Chat Experience:** Streamed responses (server-sent events or WebSockets).
- **Actionable Plans:** Interactive UI for generated training/diet plans.

### 5.3 Interactive Reports
- **Heatmaps:** Overlaying detection coordinates on a virtual pitch.
- **Video Highlight Reel:** Deep-linking analysis events to specific video timestamps.

## 6. Testing Strategy
- **Unit Testing:** Business logic (Usecases, ViewModels).
- **Widget Testing:** UI components.
- **Integration Testing:** Full flow (Auth -> Upload -> Report).
- **CI/CD:** GitHub Actions / Bitrise for automated builds.
