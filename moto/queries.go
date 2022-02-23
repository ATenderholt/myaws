package moto

import (
	"context"
	"database/sql"
	"errors"
	"myaws/database"
	"myaws/log"
	"myaws/moto/types"
)

func InsertRequest(ctx context.Context, db *database.Database, apiRequest *types.ApiRequest) error {
	log.Info("Inserting API request for %s ...", apiRequest.Service)
	log.Debug("Authorization: %s", apiRequest.Authorization)
	log.Debug("Payload: %s", apiRequest.Payload)

	id, err := db.InsertOne(
		ctx,
		`INSERT INTO moto_request (service, method, path, authorization, content_type, payload)
					VALUES (?, ?, ?, ?, ?, ?)
		`,
		apiRequest.Service,
		apiRequest.Method,
		apiRequest.Path,
		apiRequest.Authorization,
		apiRequest.ContentType,
		apiRequest.Payload,
	)

	if err != nil {
		msg := errorMessage(apiRequest, err)
		log.Error(msg)
		return errors.New(msg)
	}

	log.Info("Inserted API request #%d for %s.", id, apiRequest.Service)
	return nil
}

func FindAllRequests(ctx context.Context, db *database.Database) (<-chan types.ApiRequest, <-chan bool, <-chan error) {
	results := make(chan types.ApiRequest)
	errs := make(chan error)
	done := make(chan bool)

	go func() {
		defer close(results)
		defer close(errs)
		defer close(done)

		rows, err := db.QueryContext(
			ctx,
			`SELECT id, service, method, path, authorization, content_type, payload FROM moto_request ORDER BY id`,
		)

		if err != nil {
			msg := log.Error("unable to query all moto api requests: %v", err)
			errs <- errors.New(msg)
			return
		}

		for rows.Next() {
			var result types.ApiRequest
			err := rows.Scan(
				&result.ID,
				&result.Service,
				&result.Method,
				&result.Path,
				&result.Authorization,
				&result.ContentType,
				&result.Payload,
			)

			if err != nil {
				msg := log.Error("unable to extract results when querying all moto api requests: %v", err)
				errs <- errors.New(msg)
				return
			}

			results <- result
		}

		if rows.Err() == sql.ErrNoRows {
			done <- true
			return
		}

		errs <- rows.Err()
	}()

	return results, done, errs
}
