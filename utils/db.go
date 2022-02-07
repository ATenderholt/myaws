package utils

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"myaws/config"
	"path/filepath"
)

func CreateConnection() *sql.DB {
	settings := config.GetSettings()
	dbPath := filepath.Join(settings.GetDataPath(), "db.sqlite3")
	connStr := fmt.Sprintf("file:%s", dbPath)

	db, err := sql.Open("sqlite3", connStr)
	if err != nil {
		panic(SqlError{"unable to open database", err})
	}

	err = db.Ping()
	if err != nil {
		panic(SqlError{"unable to ping db", err})
	}

	return db
}
