package queries

import (
	"context"
	"database/sql"
	"errors"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/database"
	"myaws/log"
)

func RuntimeExistsByName(ctx context.Context, db *database.Database, runtime aws.Runtime) (bool, error) {
	log.Info("Querying for Lambda Runtime %s ...", runtime)
	var id int
	var name string
	err := db.QueryRowContext(
		ctx,
		`SELECT id, name from lambda_runtime WHERE name = ?`,
		runtime,
	).Scan(&id, &name)

	switch {
	case err == sql.ErrNoRows:
		log.Info("... not found")
		return false, nil
	case err != nil:
		msg := log.Error("error when querying runtime %s: %v", runtime, err)
		return false, errors.New(msg)
	}

	log.Info("... found %s with id=%s", name, id)
	return true, nil
}

func RuntimeIDsByNames(ctx context.Context, db *database.Database, runtimes []aws.Runtime) (map[aws.Runtime]int, error) {
	results := make(map[aws.Runtime]int, len(runtimes))
	var resultError error = nil
	for _, runtime := range runtimes {
		var id int
		var name string
		err := db.QueryRowContext(
			ctx,
			`SELECT id, name from lambda_runtime WHERE name = ?`,
			runtime,
		).Scan(&id, &name)

		switch {
		case err == sql.ErrNoRows:
			log.Error("unable to find Layer Runtime %s", runtime)
			resultError = sql.ErrNoRows
			results[runtime] = -1
		case err != nil:
			msg := log.Error("error when querying runtime %s: %v", runtime, err)
			return nil, errors.New(msg)
		default:
			log.Info("Found Layer Runtime id=%d name=%s", id, name)
			results[runtime] = id
		}
	}

	return results, resultError
}
