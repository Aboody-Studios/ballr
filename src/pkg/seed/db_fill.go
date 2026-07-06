package seed

import (
	"context"
	"log"

	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	coachinfra "github.com/Aboody-Studios/ballr/src/internal/coach/infrastructure"
	userdomain "github.com/Aboody-Studios/ballr/src/internal/identity/domain"
	userinfra "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	matchdomain "github.com/Aboody-Studios/ballr/src/internal/match/domain"
	matchinfra "github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
	progressdomain "github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	progressinfra "github.com/Aboody-Studios/ballr/src/internal/progress/infrastructure"
	gofakeit "github.com/brianvoe/gofakeit/v7"
)

func FillDB(coachRepo coachdomain.CoachRepo, progressRepo progressdomain.ProgressRepository,
	matchRepo matchdomain.MatchRepository, userRepo userdomain.UserRepository,
	achievementRepo progressdomain.AchievementRepository, eventLogRepo progressdomain.EventLogRepository) error {
	for range 50 {
		user := userinfra.User{
			ID:         gofakeit.UUID(),
			Email:      gofakeit.Email(),
			Username:   gofakeit.Username(),
			FullName:   gofakeit.Name(),
			BirthDate:  gofakeit.Date(),
			CreatedAt:  gofakeit.Date(),
			AvatarURL:  "link",
			Footedness: "left",
			Timezone:   "EUR",
		}

		userDomain := userinfra.FromUserInfraToDomain(user)

		if err := userRepo.Create(context.Background(), userDomain); err != nil {
			log.Printf("Failed to seed user: %v", err)
			continue
		}

		progress := progressinfra.Progress{
			ID:            gofakeit.UUID(),
			UserID:        user.ID,
			TotalPoints:   50,
			CurrentStreak: 4,
			LastActive:    gofakeit.Date(),
			CreatedAt:     gofakeit.Date(),
			UpdatedAt:     gofakeit.Date(),
		}
		progressDomain := progressinfra.FromProgressInfraToDomain(progress)

		if err := progressRepo.Save(context.Background(), progressDomain); err != nil {
			return err
		}

		achiev := progressinfra.Achievement{
			ID:          gofakeit.UUID(),
			ProgressID:  progress.ID,
			UserID:      user.ID,
			Type:        "FIRST_UPLOAD",
			UnlockedAt:  gofakeit.Date(),
			PointsValue: 100,
			Badge:       true,
		}
		achievDomain := progressinfra.FromAchievInfraToDomain(achiev)

		if err := achievementRepo.Save(context.Background(), achievDomain); err != nil {
			return err
		}

		eventLog := progressinfra.EventLog{
			UserID:        user.ID,
			Type:          "MATCH_UPLOADED",
			PointsAwarded: 100,
			ID:            gofakeit.UUID(),
			Timestamp:     gofakeit.Date(),
		}

		eventLogDomain := progressinfra.FromEventlogInfraToDomain(eventLog)

		if err := eventLogRepo.Save(context.Background(), eventLogDomain); err != nil {
			return err
		}

		/*deviceInfo := progressinfra.DeviceInfo{
			ID: gofakeit.ID(),
			UserID: user.ID,
			DeviceToken: "tokennnnnnnnnnnnnnnnnnnnnnnn",
		}*/

		shirtNumbers := []uint{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25}

		match := matchinfra.Match{
			ID:             gofakeit.UUID(),
			UserID:         user.ID,
			ShirtNumber:    gofakeit.RandomUint(shirtNumbers),
			PositionPlayed: "LB",
			VideoURL:       "url",
			Status:         "COMPLETED",
			AnalysisFlag:   true,
			CreatedAt:      gofakeit.Date(),
			UpdatedAt:      gofakeit.Date(),
		}

		matchDomain := matchinfra.FromMatchInfraToDomain(match)

		if err := matchRepo.Save(context.Background(), matchDomain); err != nil {
			return err
		}

		chatMessage := coachinfra.ChatMessage{
			ID:        gofakeit.UUID(),
			UserID:    user.ID,
			Role:      "user",
			Content:   "lolololololololo",
			CreatedAt: gofakeit.Date(),
		}

		chatMessageDomain := coachinfra.FromChatInfraToDomain(chatMessage)

		if err := coachRepo.SaveMessage(context.Background(), chatMessageDomain); err != nil {
			return err
		}
	}

	return nil
}
