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

var userD userinfra.User
var matchD matchinfra.Match
var chatMsgD coachinfra.ChatMessage
var deviceInfoD progressinfra.DeviceInfo
var progressD progressinfra.Progress
var achiD progressinfra.Achievement
var eventLogD progressinfra.EventLog


func FillDB(coachRepo coachdomain.CoachRepo, progressRepo progressdomain.ProgressRepository, 
	matchRepo matchdomain.MatchRepository, userRepo userdomain.UserRepository, 
	achievementRepo progressdomain.AchievementRepository, eventLogRepo progressdomain.EventLogRepository) error {
//TODO!: Put this in a loop and use same uuid for each foreign key
	for range 50 {
		user := userinfra.User{
			ID: gofakeit.ID(),
			Email: gofakeit.Email(),
			Username: gofakeit.Username(),
			FullName: gofakeit.Name(),
			BirthDate: gofakeit.Date(),
			CreatedAt: gofakeit.Date(),
			AvatarURL: "link",
			Footedness: "left",
			Timezone: "EUR",
		}
		
		userDomain:= userinfra.FromUserInfraToDomain(user)

		if err := userRepo.Create(context.Background(), userDomain); err != nil {
			log.Printf("Failed to seed user: %v", err)
			continue
		}

		progress := progressinfra.Progress{
			ID: gofakeit.ID(),
			UserID: user.ID,
			TotalPoints: 50,
			CurrentStreak: 4,
			LastActive: gofakeit.Date(),
			CreatedAt: gofakeit.Date(),
			UpdatedAt: gofakeit.Date(),
		}
		progressDomain := progressinfra.FromProgressInfraToDomain(progress)

		if err := progressRepo.Save(context.Background(), progressDomain); err != nil {
			return err
		}

		achiev := progressinfra.Achievement{
			ID: gofakeit.ID(),
			ProgressID: progress.ID,
			UserID: user.ID,
			Type: "FIRST_UPLOAD",
			UnlockedAt: gofakeit.Date(),
			PointsValue: 100,
			Badge: true,
		}
		achievDomain := progressinfra.FromAchievInfraToDomain(achiev)

		if err := achievementRepo.Save(context.Background(), achievDomain); err != nil {
			return err
		}

		eventLog := progressinfra.EventLog{
			UserID: user.ID,
			Type: "MATCH_UPLOADED",
			PointsAwarded: 100,
			ID: gofakeit.ID(),
			Timestamp: gofakeit.Date(),
		}

		eventLogDomain := progressinfra.FromEventlogInfraToDomain(eventLog)

		if err := eventLogRepo.Save(context.Background(), eventLogDomain); err != nil {
			return err
		}


	}

	return nil
}
