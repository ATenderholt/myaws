package moto

import (
	"context"
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
