package queries

import (
	"context"
	"database/sql"
	"fmt"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/database"
	"myaws/lambda/types"
	"myaws/log"
	"strings"
	"time"
)

const queryLatestVersionByLayerName = `
SELECT name, max(version) from lambda_layer where name = ?
`

func InsertLayer(ctx context.Context, db *database.Database, layer types.LambdaLayer, dbRuntimes *map[aws.Runtime]int) (*types.LambdaLayer, error) {

	tx, err := db.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to create transaction to add lambda layer %v: %v", layer, err)
	}

	log.Info("Inserting Lambda Layer %+v", layer)

	createdOn := time.Now()
	layerId, err := tx.InsertOne(
		ctx,
		`INSERT INTO lambda_layer (name, description, version, created_on, code_size, code_sha256)
					VALUES (?, ?, ?, ?, ?, ?)
		`,
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

	stmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO lambda_layer_runtime (lambda_layer_id, lambda_runtime_id)
					VALUES (?, ?)
		`,
	)

	if err != nil {
		msg := tx.Rollback("unable to prepare statement for inserting layer runtimes for %s: %v", layer.Name, err)
		return nil, fmt.Errorf(msg)
	}

	for runtimeName, runtimeId := range *dbRuntimes {
		log.Debug("Trying to insert runtime %s for layer %s", runtimeName, layer.Name)
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

	result := types.LambdaLayer{
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

func LayerByName(ctx context.Context, db *database.Database, name string) ([]types.LambdaLayer, error) {
	var results []types.LambdaLayer
	rows, err := db.QueryContext(
		ctx, `SELECT ll.id, ll.name, ll.description, ll.version, ll.created_on, GROUP_CONCAT(r.name) AS runtimes,
    					ll.code_size, ll.code_sha256
					FROM lambda_runtime AS r
					INNER JOIN lambda_layer_runtime llr ON r.id = llr.lambda_runtime_id
					INNER JOIN lambda_layer ll ON llr.lambda_layer_id = ll.id
					WHERE ll.name = ?
					GROUP BY llr.lambda_layer_id;
		`,
		name,
	)

	switch {
	case err == sql.ErrNoRows:
		return results, nil
	case err != nil:
		return nil, fmt.Errorf("problem querying for all versions for layer %s: %v", name, err)
	}

	for rows.Next() {
		var result types.LambdaLayer
		var createdOn int64
		var runtimes string
		err := rows.Scan(&result.ID, &result.Name, &result.Description, &result.Version, &createdOn, &runtimes,
			&result.CodeSize, &result.CodeSha256)

		if err != nil {
			return results, fmt.Errorf("problem parsing results when querying all versions for layer %s: %v", name, err)
		}

		result.CreatedOn = time.UnixMilli(createdOn).Format("2006-01-02T15:04:05.999-0700")
		result.CompatibleRuntimes = stringToRuntimes(runtimes)

		log.Info("got row when querying lambda layer %s: %+v", name, result)
		results = append(results, result)
	}

	log.Info("returning results: %+v", results)
	return results, nil
}

func LayerByNameAndVersion(ctx context.Context, db *database.Database, name string, version int) (types.LambdaLayer, error) {
	var result types.LambdaLayer
	var createdOn int64
	var runtimes string

	log.Info("Querying for Layer by Name and Version: %s / %d ...", name, version)

	err := db.QueryRowContext(
		ctx,
		`SELECT ll.id, ll.name, ll.description, ll.version, ll.created_on, GROUP_CONCAT(r.name) AS runtimes,
       				ll.code_size, ll.code_sha256
				FROM lambda_runtime AS r
				INNER JOIN lambda_layer_runtime llr ON r.id = llr.lambda_runtime_id
				INNER JOIN lambda_layer ll ON llr.lambda_layer_id = ll.id
				WHERE ll.name = ? AND ll.version = ?
				GROUP BY llr.lambda_layer_id;
		`,
		name,
		version,
	).Scan(
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

func stringToRuntimes(runtime string) []aws.Runtime {
	log.Debug("converting %s to list of runtimes", runtime)
	split := strings.Split(runtime, ",")
	runtimes := make([]aws.Runtime, len(split))
	for i, value := range split {
		runtimes[i] = aws.Runtime(value)
	}

	return runtimes
}

func LatestLayerByName(ctx context.Context, db *database.Database, name string) (int, error) {
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
