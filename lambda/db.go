package lambda

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"log"
	"myaws/database"
	"strings"
	"time"
)

var Migrations = []database.Migration{
	{
		Service:     "Lambda",
		Description: "Create Layer Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_layer (
					id           integer primary key autoincrement,
					name         text not null,
					description  text not null,
					version      integer not null,
					created_on   integer not null,
					code_size	 integer not null,
					code_sha256  text not null
				);
		`,
	},
	{
		Service:     "Lambda",
		Description: "Create Runtime Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_runtime (
					id      integer primary key autoincrement,
					name	text not null unique
				);
			
				INSERT OR IGNORE INTO lambda_runtime (name) VALUES
				('python3.6'),
				('python3.7'),
				('python3.8');
		`,
	},
	{
		Service:     "Lambda",
		Description: "Create Layer Runtime Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_layer_runtime (
					id					integer primary key autoincrement,
					lambda_layer_id		integer,
					lambda_runtime_id	integer,
					FOREIGN KEY(lambda_layer_id) REFERENCES lambda_layer(id),
					FOREIGN	KEY(lambda_runtime_id) REFERENCES lambda_runtime(id)
				);
		`,
	},
}

const insertLayer = `
INSERT INTO lambda_layer (name, description, version, created_on, code_size, code_sha256)
VALUES (?, ?, ?, ?, ?, ?)
`

const queryLatestVersionByLayerName = `
SELECT name, max(version) from lambda_layer where name = ?
`

const queryAllVersionsByLayerName = `
SELECT ll.id, ll.name, ll.description, ll.version, ll.created_on, GROUP_CONCAT(r.name) AS runtimes,
    ll.code_size, ll.code_sha256
FROM lambda_runtime AS r
INNER JOIN lambda_layer_runtime llr ON r.id = llr.lambda_runtime_id
INNER JOIN lambda_layer ll ON llr.lambda_layer_id = ll.id
WHERE ll.name = ?
GROUP BY llr.lambda_layer_id;
`

const queryRuntime = `
SELECT id, name from lambda_runtime WHERE name = ?
`

const insertLayerRuntime = `
INSERT INTO lambda_layer_runtime (lambda_layer_id, lambda_runtime_id)
VALUES (?, ?)
`

func addLayer(ctx context.Context, db *database.Database, layer LambdaLayer) (*LambdaLayer, error) {
	dbRuntimes, err := getLayerRuntimes(ctx, db, layer.CompatibleRuntimes)
	switch {
	case err == sql.ErrNoRows:
		return nil, fmt.Errorf("unable to find all expected runtimes: %v", err)
	case err != nil:
		return nil, fmt.Errorf("error when adding runtime: %v", err)
	}

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create transaction to add lambda layer %v: %v", layer, err)
	}

	log.Printf("Inserting lambda layer %+v", layer)

	createdOn := time.Now()
	layerId, err := tx.InsertOne(ctx, insertLayer,
		layer.Name,
		layer.Description,
		layer.Version,
		createdOn.UnixMilli(),
		layer.CodeSize,
		layer.CodeSha256,
	)

	if err != nil {
		return nil, fmt.Errorf("unable to insert layer %s: %v", layer.Name, err)
	}

	stmt, err := tx.PrepareContext(ctx, insertLayerRuntime)
	if err != nil {
		msg := tx.Rollback("unable to prepare statement for inserting layer runtimes for %s: %v", layer.Name, err)
		return nil, fmt.Errorf(msg)
	}

	for runtimeName, runtimeId := range dbRuntimes {
		log.Printf("Trying to insert runtime %s for layer %s", runtimeName, layer.Name)
		_, err := stmt.ExecContext(ctx, layerId, runtimeId)
		if err != nil {
			msg := tx.Rollback("unable to insert runtime %s for layer %s: %v", runtimeName, layer.Name, err)
			return nil, fmt.Errorf(msg)
		}
	}

	err = tx.Commit()
	if err != nil {
		return nil, fmt.Errorf("unable to commit layer %s: %v", layer.Name, err)
	}

	result := LambdaLayer{
		ID:                 layerId,
		Name:               layer.Name,
		Version:            layer.Version,
		Description:        layer.Description,
		CreatedOn:          createdOn.Format("2006-01-02T15:04:05.999-0700"),
		CompatibleRuntimes: layer.CompatibleRuntimes,
		CodeSize:           layer.CodeSize,
		CodeSha256:         layer.CodeSha256,
	}

	return &result, nil
}

func getLayerRuntimes(ctx context.Context, db *database.Database, runtimes []types.Runtime) (map[types.Runtime]int, error) {
	results := make(map[types.Runtime]int, len(runtimes))
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

func getAllLayerVersions(ctx context.Context, db *database.Database, name string) ([]LambdaLayer, error) {
	var results []LambdaLayer
	rows, err := db.QueryContext(ctx, queryAllVersionsByLayerName, name)
	switch {
	case err == sql.ErrNoRows:
		return results, nil
	case err != nil:
		return nil, fmt.Errorf("problem querying for all versions for layer %s: %v", name, err)
	}

	for rows.Next() {
		var result LambdaLayer
		var createdOn int64
		var runtimes string
		err := rows.Scan(&result.ID, &result.Name, &result.Description, &result.Version, &createdOn, &runtimes,
			&result.CodeSize, &result.CodeSha256)

		if err != nil {
			return results, fmt.Errorf("problem parsing results when querying all versions for layer %s: %v", name, err)
		}

		result.CreatedOn = time.UnixMilli(createdOn).Format("2006-01-02T15:04:05.999-0700")
		result.CompatibleRuntimes = stringToRuntimes(runtimes)

		log.Printf("got row when querying lambda layer %s: %+v", name, result)
		results = append(results, result)
	}

	log.Printf("returning results: %+v", results)
	return results, nil
}

func getLayerVersion(ctx context.Context, db *database.Database, name string, version int) (LambdaLayer, error) {
	var result LambdaLayer
	var createdOn int64
	var runtimes string

	query := `
SELECT ll.id, ll.name, ll.description, ll.version, ll.created_on, GROUP_CONCAT(r.name) AS runtimes,
       ll.code_size, ll.code_sha256
FROM lambda_runtime AS r
INNER JOIN lambda_layer_runtime llr ON r.id = llr.lambda_runtime_id
INNER JOIN lambda_layer ll ON llr.lambda_layer_id = ll.id
WHERE ll.name = ? AND ll.version = ?
GROUP BY llr.lambda_layer_id;
`
	err := db.QueryRowContext(ctx, query, name, version).Scan(
		&result.ID,
		&result.Name,
		&result.Description,
		&result.Version,
		&createdOn,
		&runtimes,
		&result.CodeSize,
		&result.CodeSha256,
	)

	if err != nil {
		return result, fmt.Errorf("problem parsing results when querying version %d for layer %s: %v",
			version, name, err)
	}

	result.CreatedOn = time.UnixMilli(createdOn).Format("2006-01-02T15:04:05.999-0700")
	result.CompatibleRuntimes = stringToRuntimes(runtimes)

	return result, nil
}

func stringToRuntimes(runtime string) []types.Runtime {
	log.Printf("converting %s to list of runtimes", runtime)
	split := strings.Split(runtime, ",")
	runtimes := make([]types.Runtime, len(split))
	for i, value := range split {
		runtimes[i] = types.Runtime(value)
	}

	return runtimes
}

func getLatestLayerVersion(ctx context.Context, db *database.Database, name string) (int, error) {
	var dbName sql.NullString
	var dbVersion sql.NullInt32
	err := db.QueryRowContext(ctx, queryLatestVersionByLayerName, name).Scan(&dbName, &dbVersion)
	if err != nil {
		return -1, err
	}

	if dbName.Valid && dbVersion.Valid {
		return int(dbVersion.Int32), nil
	}

	return -1, sql.ErrNoRows
}
