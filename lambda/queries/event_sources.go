package queries

import (
	"context"
	"errors"
	"myaws/database"
	"myaws/lambda/types"
	"myaws/log"
)

func SaveEventSource(ctx context.Context, db *database.Database, eventSource types.EventSource) error {
	_, err := db.InsertOne(
		ctx,
		`INSERT INTO lambda_event_source (uuid, enabled, arn, function_id, batch_size, last_modified_on)
					VALUES (?, ?, ?, ?, ?, ?)
		`,
		eventSource.UUID.String(),
		eventSource.Enabled,
		eventSource.Arn,
		eventSource.Function.ID,
		eventSource.BatchSize,
		eventSource.LastModified,
	)

	if err != nil {
		msg := log.Error("unable to insert eventSource %+v: %v", eventSource, err)
		return errors.New(msg)
	}

	return nil
}
