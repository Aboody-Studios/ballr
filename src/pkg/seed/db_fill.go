package seed

import (
	coachdomain "github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	userinfra "github.com/Aboody-Studios/ballr/src/internal/identity/infrastructure"
	matchinfra "github.com/Aboody-Studios/ballr/src/internal/match/infrastructure"
	progressinfra "github.com/Aboody-Studios/ballr/src/internal/progress/infrastructure"
	gofakeit "github.com/brianvoe/gofakeit/v7"
)

var userD userinfra.User
var progressD progressinfra.Progress
var matchD matchinfra.Match
var chatMsgD coachdomain.ChatMessage
var deviceInfoD progressinfra.DeviceInfo
var achiD progressinfra.Achievement
var eventLogD progressinfra.EventLog


func FillDB(lol any) error {
//TODO!: Put this in a loop and use same uuid for each foreign key

	if err := gofakeit.Struct(&userD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&progressD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&matchD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&chatMsgD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&deviceInfoD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&achiD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&eventLogD); err != nil {
		return err
	}

	return nil
}
