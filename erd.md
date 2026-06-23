# Cermin ERD

This ERD is based on the current Firestore structure, security rules, and TypeScript models in the app.

```mermaid
erDiagram
    USER ||--o{ JOURNAL_ENTRY : owns
    USER ||--o{ LIFE_CHAPTER : owns
    USER ||--o{ DAILY_SUMMARY : owns
    USER ||--o| MOOD_FINGERPRINT : embeds

    JOURNAL_ENTRY ||--o| REFLECTION_DATA : embeds
    REFLECTION_DATA ||--o{ EMOTION_SCORE : contains

    LIFE_CHAPTER ||--o{ EMOTION_SCORE : summarizes
    LIFE_CHAPTER }o--o{ JOURNAL_ENTRY : references_pivotal_entries

    DAILY_SUMMARY }o--o{ JOURNAL_ENTRY : summarizes_by_day

    USER {
        string id PK "Firebase Auth uid / users/{userId}"
        string userName
        map fingerprint "optional embedded mood fingerprint"
    }

    JOURNAL_ENTRY {
        string id PK "entries/{entryId}"
        string userId FK
        string content
        string createdAt
        map reflection "optional embedded reflection"
        boolean isCrisis
    }

    REFLECTION_DATA {
        string validation
        string[] summary
        string growthPrompt
        string dominantEmotion
        string[] hiddenLanguage
    }

    EMOTION_SCORE {
        string label "Senang | Sedih | Marah | Cemas | Tenang | Lelah | Harapan | Lainnya"
        number percentage
    }

    LIFE_CHAPTER {
        string id PK "chapters/{chapterId}"
        string userId FK
        number number
        string title
        string start_date
        string end_date
        number total_days
        number total_entries
        string narrative
        string[] dominant_themes
        string[] pivotal_entries "JournalEntry ids"
        string generated_at
        boolean is_current
    }

    DAILY_SUMMARY {
        string id PK "dailySummaries/{summaryId}"
        string userId FK
        string summary
        number entryCountAtGeneration
        string lastGeneratedAt
    }

    MOOD_FINGERPRINT {
        string signature_emotion
        string description
        string[] patterns
        string advice
    }
```

## Firestore Paths

```text
users/{userId}
users/{userId}/entries/{entryId}
users/{userId}/chapters/{chapterId}
users/{userId}/dailySummaries/{summaryId}
```

## Relationship Notes

- `User` is the root owner for all persisted app data.
- `JournalEntry`, `LifeChapter`, and `DailySummary` are stored as subcollections under `users/{userId}`.
- `ReflectionData` is embedded inside a journal entry, not stored as a separate collection.
- `EmotionScore` is embedded inside `ReflectionData` and `LifeChapter`.
- `MoodFingerprint` is modeled as optional embedded data on the user document, although the current UI also caches fingerprint data in `localStorage`.
- `LifeChapter.pivotal_entries` stores journal entry IDs, creating a logical many-to-many reference to `JournalEntry`.
- `DailySummary` summarizes entries for a day, but it does not currently store explicit entry IDs.
