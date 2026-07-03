package seed

import (
	gofakeit "github.com/brianvoe/gofakeit/v7"
)




func FillDB(lol any) error {

	if err := gofakeit.Struct(&userD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&progressD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&matchD); err != nil {
		return err
	}

	if err := gofakeit.Struct(&chatMsg); err != nil {
		return err
	}

	if err := gofakeit.Struct(&userD); err != nil {
		return err
	}

	return nil
}
