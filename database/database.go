package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"myaws/config"
	"myaws/log"
	"path/filepath"
)

type Database struct {
	wrapped *sql.DB
}

func CreateConnection() *Database {
	dbPath := filepath.Join(config.GetDataPath(), "db.sqlite3")
	connStr := fmt.Sprintf("file:%s", dbPath)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		log.Panic("unable to open database: %v", err)
	}

	err = db.Ping()
	if err != nil {
		log.Panic("unable to ping database: %v", err)
	}

	return &Database{db}
}

func (db *Database) Close() {
	err := db.wrapped.Close()
	if err != nil {
		log.Panic("unable to close database: %v", err)
	}
}

func (db *Database) BeginTx(ctx context.Context) (*Transaction, error) {
	options := sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}
	tx, err := db.wrapped.BeginTx(ctx, &options)
	if err != nil {
		return nil, err
	}

	return &Transaction{tx}, nil
}

func (db *Database) InsertOne(ctx context.Context, query string, args ...interface{}) (int64, error) {
	insert, err := db.wrapped.ExecContext(ctx, query, args...)
	if err != nil {
		debug := buildDebug(query, args)
		msg := log.Error("unable to insert %s: %s", debug, err)
		return -1, errors.New(msg)
	}

	count, err := insert.RowsAffected()
	if err != nil {
		msg := log.Error("unexpected error when inserting: %v", err)
		return -1, errors.New(msg)
	}

	if count != 1 {
		msg := log.Error("expected only 1 insert but got %d", count)
		return -1, errors.New(msg)
	}

	id, err := insert.LastInsertId()
	if err != nil {
		msg := log.Error("unexpected error when inserting")
		return -1, errors.New(msg)
	}

	return id, nil
}

func (db *Database) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.wrapped.Exec(query, args...)
}

func (db *Database) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.wrapped.QueryContext(ctx, query, args...)
}

func (db *Database) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.wrapped.QueryRow(query, args...)
}

func (db *Database) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.wrapped.QueryRowContext(ctx, query, args...)
}
