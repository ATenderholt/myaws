package queries

import (
	"context"
	"errors"
	"myaws/database"
	"myaws/log"
	"myaws/sqs/types"
)

func SaveQueue(ctx context.Context, db *database.Database, queue *types.Queue) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		msg := log.Error("Unable to begin transaction to save %s: %v", queue.Name, err)
		return errors.New(msg)
	}

	attributeStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO sqs_queue_attribute (name, key, value) VALUES (?, ?, ?)`,
	)
	defer attributeStmt.Close()

	for key, value := range queue.Attributes {
		_, err = attributeStmt.ExecContext(ctx, queue.Name, key, value)
		if err != nil {
			msg := tx.Rollback("Unable to insert attribute %s for %s: %v", key, queue.Name, err)
			return errors.New(msg)
		}
	}

	tagStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO sqs_queue_tag (name, key, value) VALUES (?, ?, ?)`,
	)
	defer tagStmt.Close()

	for key, value := range queue.Tags {
		_, err = tagStmt.ExecContext(ctx, queue.Name, key, value)
		if err != nil {
			msg := tx.Rollback("Unable to insert tag %s for %s: %v", key, queue.Name, err)
			return errors.New(msg)
		}
	}

	err = tx.Commit()
	if err != nil {
		msg := log.Error("Unable to commit transaction to save %s: %v", queue.Name, err)
		return errors.New(msg)
	}

	return nil
}

func LoadQueue(ctx context.Context, db *database.Database, name string) (*types.Queue, error) {
	queue := types.NewQueue(name)

	rows, err := db.QueryContext(
		ctx,
		`SELECT key, value from sqs_queue_attribute WHERE name = ?`,
		name,
	)

	if err != nil {
		msg := log.Error("Unable to load attributes for queue %s: %v", name, err)
		return nil, errors.New(msg)
	}

	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			msg := log.Error("Unable to scan attributes for queue %s: %v", name, err)
			return nil, errors.New(msg)
		}

		queue.Attributes[key] = value
	}

	rows, err = db.QueryContext(
		ctx,
		`SELECT key, value from sqs_queue_tag WHERE name = ?`,
		name,
	)

	if err != nil {
		msg := log.Error("Unable to load tags for queue %s: %v", name, err)
		return nil, errors.New(msg)
	}

	for rows.Next() {
		var key, value string
		err := rows.Scan(&key, &value)
		if err != nil {
			msg := log.Error("Unable to scan tag for queue %s: %v", name, err)
			return nil, errors.New(msg)
		}

		queue.Tags[key] = value
	}

	return queue, nil
}
