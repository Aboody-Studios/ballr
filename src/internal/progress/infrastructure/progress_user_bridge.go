package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/progress/domain"
	"gorm.io/gorm"
)

type ProgressUserBridgeDB struct {
	*gorm.DB
}

func (pub *ProgressUserBridgeDB) GetEightPmAndTrainingDayUsers(ctx context.Context) ([]domain.NotificationTarget, error) {
	var targets []domain.NotificationTarget

	tx := pub.DB.WithContext(ctx).Table("users").Select("id, notification_token").Where(
		`timezone IS NOT NULL 
		AND EXTRACT(HOUR FROM (NOW() AT TIME ZONE timezone)) = 20
		AND training_days @> jsonb_build_array(EXTRACT(DOW FROM (NOW() AT TIME ZONE timezone)))`,
	).Scan(&targets)

	if tx.Error != nil {
		return nil, tx.Error
	}

	return targets, nil
}

func (pub *ProgressUserBridgeDB) GetUsernames(ctx context.Context, userIDs []string) (map[string]string, error) {
	type userRow struct {
		ID       string
		Username string
	}
	var userRowSlice = make([]userRow, 0, len(userIDs))

	obj := pub.WithContext(ctx).Table("users").Where("id IN ?", userIDs).Select("id, username").Find(&userRowSlice)
	if obj.Error != nil {
		return nil, obj.Error
	}

	IDtoUsername := make(map[string]string, len(userIDs))

	for _, currUserRow := range userRowSlice {
		IDtoUsername[currUserRow.ID] = currUserRow.Username
	}

	return IDtoUsername, nil
}
