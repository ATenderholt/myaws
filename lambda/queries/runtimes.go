package queries

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"myaws/database"
	"myaws/log"
)

func RuntimeIDsByNames(ctx context.Context, db *database.Database, runtimes []types.Runtime) (map[types.Runtime]int, error) {
	results := make(map[types.Runtime]int, len(runtimes))
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
			return nil, fmt.Errorf("error when querying runtime %s", runtime)
		default:
			log.Info("Found Layer Runtime id=%d name=%s", id, name)
			results[runtime] = id
		}
	}

	return results, resultError
}
