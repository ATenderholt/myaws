package utils

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"myaws/config"
	"path/filepath"
)

func CreateConnection() *sql.DB {
	settings := config.GetSettings()
	dbPath := filepath.Join(settings.GetDataPath(), "database.sqlite3")
	connStr := fmt.Sprintf("file:%s", dbPath)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		panic(SqlError{"unable to open database", err})
	}

	err = db.Ping()
	if err != nil {
		panic(SqlError{"unable to ping database", err})
	}

	return db
}

func InsertOne(tx *sql.Tx, ctx context.Context, query string, args ...interface{}) (int64, error) {
	insert, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return -1, SqlError{"unable to insert", err}
	}

	count, err := insert.RowsAffected()
	if err != nil {
		msg := rollBack(tx, "unexpected error when inserting")
		return -1, SqlError{msg, err}
	}

	if count != 1 {
		msg := rollBack(tx, fmt.Sprintf("expected only 1 insert but got %d", count))
		return -1, SqlError{msg, nil}
	}

	id, err := insert.LastInsertId()
	if err != nil {
		msg := rollBack(tx, "unexpected error when inserting")
		return -1, SqlError{msg, err}
	}

	return id, nil
}

func rollBack(tx *sql.Tx, msg string) string {
	err := tx.Rollback()
	if err != nil {
		return msg + ", couldn't rollback"
	}

	return msg
}
