# Database Schema Specification: Ballr

## 1. Overview
The database layer uses a hybrid approach:
- **PostgreSQL (SQL):** For structured relational data (Users, Progress, Metadata).
- **MongoDB or JSONB (Document):** For complex match analysis results (Event time-series).

## 2. PostgreSQL Tables

### 2.1 Users
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `email` | String | Unique email |
| `password_hash` | String | Encrypted password |
| `full_name` | String | User's name |
| `birth_date` | Date | Used for age calculation |
| `position` | Enum | GK, CB, LB, RB, CM, LW, RW, ST |
| `footedness` | Enum | Left, Right, Both |
| `goals` | Text | Primary improvement goals |
| `created_at` | Timestamp | Account creation |

### 2.2 Matches
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (Users.id) |
| `video_url` | String | Cloud storage URL |
| `shirt_number` | Integer | Player's number for ID |
| `position_played`| Enum | Position during the match |
| `status` | Enum | UPLOADING, PROCESSING, COMPLETED, FAILED |
| `upload_at` | Timestamp | |
| `metadata` | JSONB | Match date, duration, score |

### 2.3 Progress & Points
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (Users.id) |
| `total_points` | BigInt | Cumulative score |
| `current_streak` | Integer | Daily consecutive usage |
| `last_active` | Date | For streak calculation |

### 2.4 Achievements
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `user_id` | UUID | Foreign Key (Users.id) |
| `type` | String | Achievement identifier |
| `unlocked_at` | Timestamp | |

## 3. Analysis Results (Document Store)
Stored as a separate collection/table for each match analysis.

### 3.1 Match Analysis Document
```json
{
  "match_id": "uuid",
  "summary": {
    "total_distance": 10.5,
    "top_speed": 32.1,
    "pass_accuracy": 0.82
  },
  "heatmaps": {
    "overall": "base64_or_url",
    "defensive": "base64_or_url"
  },
  "events": [
    {
      "timestamp": "15:30",
      "type": "PASS",
      "result": "SUCCESS",
      "coordinates": { "x": 45, "y": 20 },
      "insight": "Great progressive pass under pressure."
    }
  ],
  "tracking_data": "compressed_coordinates_url"
}
```

### 3.2 AI Coach History
```json
{
  "user_id": "uuid",
  "conversations": [
    {
      "session_id": "uuid",
      "messages": [
        { "role": "user", "content": "How do I improve my positioning?" },
        { "role": "assistant", "content": "Based on your last match at 15:30..." }
      ]
    }
  ]
}
```

## 4. Drills Bank (Static Content)
| Field | Type | Description |
| :--- | :--- | :--- |
| `id` | UUID | Primary Key |
| `name` | String | |
| `category` | Enum | PASSING, FINISHING, etc. |
| `content` | Markdown | Steps, setup, coaching points |
| `difficulty`| Enum | BEGINNER, INTERMEDIATE, ADVANCED |
