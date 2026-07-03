package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/shared/infrastructure"
	"gorm.io/gorm"
)

//TODO!: Use infrastructure layer structs

type TransactionRepo struct {
	*gorm.DB
}

func (tr *TransactionRepo) Transact(ctx context.Context, fn func(ctx context.Context) error) error {
	if err := tr.DB.Transaction(func(tx *gorm.DB) error {
		txCtx := context.WithValue(ctx, infrastructure.TxKey{}, tx)

		if err := fn(txCtx); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}
