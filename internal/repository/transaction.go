package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// StartTransaction return a tx
func StartTransaction(db *pgxpool.Pool, ctx context.Context) (pgx.Tx, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// DeferRollback rollback the transaction
func DeferRollback(tx pgx.Tx, ctx context.Context) {
	if err := tx.Rollback(ctx); err != nil {
		zap.L().Error("Failed to rollback transaction", zap.Error(err))
	}
}

// CommitTransaction commit the transaction
func CommitTransaction(tx pgx.Tx, ctx context.Context) {
	if err := tx.Commit(ctx); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
	}
}
