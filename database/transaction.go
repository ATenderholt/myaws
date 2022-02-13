package database

import (
	"context"
	"database/sql"
	"fmt"
	"myaws/log"
)

type Transaction struct {
	wrapped *sql.Tx
}

func (tx *Transaction) Commit() error {
	return tx.wrapped.Commit()
}

func (tx *Transaction) PrepareContext(ctx context.Context, query string) (*sql.Stmt, error) {
	return tx.wrapped.PrepareContext(ctx, query)
}

func (tx *Transaction) Rollback(format string, v ...interface{}) string {
	err := tx.wrapped.Rollback()
	msg := fmt.Sprintf(format, v...)
	if err != nil {
		return msg + ", couldn't rollback"
	}

	return msg
}

func (tx *Transaction) InsertOne(ctx context.Context, query string, args ...interface{}) (int64, error) {
	insert, err := tx.wrapped.ExecContext(ctx, query, args...)
	if err != nil {
		debug := buildDebug(query, args)
		log.Error("unable to insert %s: %s", debug, err)
		return -1, fmt.Errorf("unable to insert: %v", err)
	}

	count, err := insert.RowsAffected()
	if err != nil {
		msg := tx.Rollback("unexpected error when inserting")
		log.Error(msg)
		return -1, fmt.Errorf(msg)
	}

	if count != 1 {
		msg := tx.Rollback("expected only 1 insert but got %d", count)
		log.Error(msg)
		return -1, fmt.Errorf(msg)
	}

	id, err := insert.LastInsertId()
	if err != nil {
		msg := tx.Rollback("unexpected error when inserting")
		log.Error(msg)
		return -1, fmt.Errorf(msg)
	}

	return id, nil
}
