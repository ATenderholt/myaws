package queries

import (
	"context"
	"database/sql"
	"errors"
	"github.com/docker/distribution/uuid"
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

func LoadEventSource(ctx context.Context, db *database.Database, id string) (*types.EventSource, error) {
	log.Info("Loading Event Source %s ...", id)

	var err error
	var eventSource types.EventSource
	eventSource.UUID, err = uuid.Parse(id)
	if err != nil {
		msg := log.Error("Unable to parse Event Source id %s: %v", id, err)
		return nil, errors.New(msg)
	}

	row := db.QueryRowContext(
		ctx,
		`SELECT enabled, arn, function_id, batch_size, last_modified_on FROM lambda_event_source WHERE uuid=?`,
		id,
	)

	var functionId int64
	err = row.Scan(
		&eventSource.Enabled,
		&eventSource.Arn,
		&functionId,
		&eventSource.BatchSize,
		&eventSource.LastModified,
	)

	switch {
	case err == sql.ErrNoRows:
		log.Error("Event Source %s not found", id)
		return nil, nil
	case err != nil:
		msg := log.Error("Unable to find Event Source %s: %v", id, err)
		return nil, errors.New(msg)
	}

	row = db.QueryRowContext(
		ctx,
		`SELECT name, version FROM lambda_function WHERE id=? ORDER BY version DESC LIMIT 1`,
		functionId,
	)

	var function types.Function
	err = row.Scan(
		&function.FunctionName,
		&function.Version,
	)

	if err != nil {
		msg := log.Error("Unable to find Function %d for Event Source %s: %v", functionId, id, err)
		return nil, errors.New(msg)
	}

	eventSource.Function = &function

	return &eventSource, nil
}
