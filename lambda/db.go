package lambda

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"myaws/utils"
)

const createLayerTable = `
CREATE TABLE IF NOT EXISTS lambda_layer (
	id           integer primary key autoincrement,
    name         text not null,
    description  text not null,
	version      integer not null,
	created_on   integer not null
);
`

const insertLayer = `
INSERT INTO lambda_layer (name, description, version, created_on)
VALUES (?, ?, ?, ?)
`

const createRuntimeTable = `
CREATE TABLE IF NOT EXISTS lambda_runtime (
	id      integer primary key autoincrement,
	name	text not null unique
);

INSERT OR IGNORE INTO lambda_runtime (name) VALUES
('python3.6'),
('python3.7'),
('python3.8');
`

const queryRuntime = `
SELECT id, name from lambda_runtime WHERE name = ?
`

const createLayerRuntimeTable = `
CREATE TABLE IF NOT EXISTS lambda_layer_runtime (
	id					integer primary key autoincrement,
	lambda_layer_id		integer,
	lambda_runtime_id	integer,
	FOREIGN KEY(lambda_layer_id) REFERENCES lambda_layer(id),
	FOREIGN	KEY(lambda_runtime_id) REFERENCES lambda_runtime(id)
);
`

const insertLayerRuntime = `
INSERT INTO lambda_layer_runtime (lambda_layer_id, lambda_runtime_id)
VALUES (?, ?)
`

func createConnection(ctx context.Context) *sql.DB {
	db := utils.CreateConnection()
	_, err := db.ExecContext(ctx, createLayerTable)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_layer table", Err: err})
	}

	_, err = db.ExecContext(ctx, createRuntimeTable)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_runtime table", Err: err})
	}

	_, err = db.ExecContext(ctx, createLayerRuntimeTable)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_layer_runtime table", Err: err})
	}

	return db
}

var txWriteOptions = sql.TxOptions{Isolation: sql.LevelDefault, ReadOnly: false}

func addLayer(ctx context.Context, db *sql.DB, layer LambdaLayer, runtimes []Runtime) error {
	dbRuntimes, err := getLayerRuntimes(ctx, db, runtimes)
	switch {
	case err == sql.ErrNoRows:
		return fmt.Errorf("unable to find all expected runtimes: %v", err)
	case err != nil:
		return fmt.Errorf("error when adding runtime: %v", err)
	}

	tx, err := db.BeginTx(ctx, &txWriteOptions)
	if err != nil {
		return fmt.Errorf("unable to create transaction to add lambda layer %v: %v", layer, err)
	}

	log.Printf("Inserting lambda layer %+v", layer)

	insert, err := tx.ExecContext(ctx, insertLayer, layer.Name, layer.Description, layer.Version, 100)
	if err != nil {
		return fmt.Errorf("unable to insert lambda layer %s: %v", layer.Name, err)
	}

	count, err := insert.RowsAffected()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unexpected error when inserting lambda layer %s: %v", layer.Name, err)
	}

	if count != 1 {
		tx.Rollback()
		return fmt.Errorf("expected only 1 insert for lambda layer %s but got %d", layer.Name, count)
	}

	layerId, err := insert.LastInsertId()
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unexpected error when inserting lambda layer %s: %v", layer.Name, err)
	}

	stmt, err := tx.PrepareContext(ctx, insertLayerRuntime)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("unable to prepare statement for inserting layer runtimes for %s: %v", layer.Name,
			err)
	}

	for runtimeName, runtimeId := range dbRuntimes {
		log.Printf("Trying to insert runtime %s for layer %s", runtimeName, layer.Name)
		_, err := stmt.ExecContext(ctx, layerId, runtimeId)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("unable to insert runtime %s for layer %s: %v", runtimeName, layer.Name, err)
		}
	}

	tx.Commit()
	return nil
}

func getLayerRuntimes(ctx context.Context, db *sql.DB, runtimes []Runtime) (map[Runtime]int, error) {
	results := make(map[Runtime]int, len(runtimes))
	var resultError error = nil
	for _, runtime := range runtimes {
		var id int
		var name string
		err := db.QueryRowContext(ctx, queryRuntime, runtime).Scan(&id, &name)
		switch {
		case err == sql.ErrNoRows:
			log.Printf("unable to find Layer Runtime %s", runtime)
			resultError = sql.ErrNoRows
			results[runtime] = -1
		case err != nil:
			return nil, fmt.Errorf("error when querying runtime %s", runtime)
		default:
			log.Printf("Found Layer Runtime id=%d name=%s", id, name)
			results[runtime] = id
		}
	}

	return results, resultError
}
