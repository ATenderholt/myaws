package lambda

import (
	"context"
	"database/sql"
	"fmt"
	"myaws/utils"
)

const layerTableCreate = `
CREATE TABLE IF NOT EXISTS lambda_layer (
	id           integer primary key autoincrement,
    name         text not null,
	version      integer not null,
	created_on   integer not null
);
`

const runtimeTableCreate = `
CREATE TABLE IF NOT EXISTS lambda_runtime (
	id      integer primary key autoincrement,
	name	text not null unique
);

INSERT OR IGNORE INTO lambda_runtime (name) VALUES
('python3.6'),
('python3.7'),
('python3.8');
`

const layerRuntimeTableCreate = `
CREATE TABLE IF NOT EXISTS lambda_layer_runtime (
	id					integer primary key autoincrement,
	lambda_layer_id		integer,
	lambda_runtime_id	integer,
	FOREIGN KEY(lambda_layer_id) REFERENCES lambda_layer(id),
	FOREIGN	KEY(lambda_runtime_id) REFERENCES lambda_runtime(id)
);
`

func createConnection(ctx context.Context) *sql.DB {
	db := utils.CreateConnection()
	_, err := db.ExecContext(ctx, layerTableCreate)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_layer table", Err: err})
	}

	_, err = db.ExecContext(ctx, runtimeTableCreate)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_runtime table", Err: err})
	}

	_, err = db.ExecContext(ctx, layerRuntimeTableCreate)
	if err != nil {
		panic(utils.SqlError{Message: "unable to create lambda_layer_runtime table", Err: err})
	}

	return db
}

var txWriteOptions = sql.TxOptions{Isolation: sql.LevelSerializable, ReadOnly: false}

func addLayer(ctx context.Context, db *sql.DB, layer LambdaLayer, runtimes []string) error {
	tx, err := db.BeginTx(ctx, &txWriteOptions)
	if err != nil {
		return fmt.Errorf("unable to creaet transaction to add lambda layer %v: %v", layer, err)
	}

	stmt, err := tx.PrepareContext(ctx)
}
