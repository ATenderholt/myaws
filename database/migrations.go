package database

import (
	"crypto/md5"
	"database/sql"
	"fmt"
	"myaws/log"
)

type Migration struct {
	Service     string
	Description string
	Query       string
}

func (migration Migration) String() string {
	return migration.Service + " - " + migration.Description
}

type Migrations struct {
	migrations []Migration
}

func (m *Migrations) AddAll(migrations []Migration) {
	for _, migration := range migrations {
		m.migrations = append(m.migrations, migration)
	}
}

func (m *Migrations) Size() int {
	return len(m.migrations)
}

const createMigrationTable = `
CREATE TABLE IF NOT EXISTS migration (
	id          integer primary key autoincrement,
	service     text not null,
	description text not null,
	hash        text not null,
	applied     boolean
)
`

func Initialize(migrations Migrations) {
	db := CreateConnection()
	defer db.Close()

	log.Info("Creating migration table if necessary")

	_, err := db.Exec(createMigrationTable)
	if err != nil {
		log.Error("Unable to create migrations.go table: %v", err)
		panic(err)
	}

	for _, migration := range migrations.migrations {
		migration.apply(db)
	}
}

func (migration Migration) apply(db *Database) {
	log.Info("Searching for Migration %v...", migration)

	rawHash := md5.Sum([]byte(migration.Query))
	hash := fmt.Sprintf("%x", rawHash)

	var dbHash string
	var dbApplied bool
	needsApplying := false
	err := db.QueryRow("SELECT hash, applied FROM migration WHERE service = ? and description = ?",
		migration.Service, migration.Description).Scan(&dbHash, &dbApplied)

	switch {
	case err == sql.ErrNoRows:
		log.Info("... Migration %v needs to be applied.", migration)
		needsApplying = true
	case err != nil:
		log.Panic("... error when searching for Migration %v: %v.", migration, err)
	}

	if !needsApplying && !dbApplied {
		msg := log.Error("... Migration %v was already attempted, but failed to apply.", migration)
		panic(msg)
	}

	if !needsApplying && hash != dbHash {
		log.Panic("Migration %v does not need to be applied by hashes are different: found %s != expected %s.",
			migration, dbHash, hash)
	}

	if !needsApplying && dbApplied {
		log.Info("... Migration %v already applied.", migration)
		return
	}

	log.Info("Applying Migration %v...", migration)

	_, err = db.Exec(migration.Query)
	applied := false
	if err != nil {
		log.Error("... unable to apply Migration %v: %v", migration, err)
	} else {
		log.Info(".... Migration %v applied.", migration)
		applied = true
	}

	_, err = db.Exec("INSERT INTO migration (service, description, hash, applied) VALUES (?, ?, ?, ?)",
		migration.Service, migration.Description, hash, applied)
	if err != nil {
		log.Panic("Unable to save Migration %v.", migration)
	}

	if !applied {
		panic("Migrations are not all applied.")
	}
}
