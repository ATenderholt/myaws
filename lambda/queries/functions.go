package queries

import (
	"context"
	"database/sql"
	"errors"
	"myaws/database"
	"myaws/lambda/types"
	"myaws/log"
)

func LatestFunctionVersionByName(ctx context.Context, db *database.Database, name *string) (int, error) {
	log.Info("Querying for Lambda Function %s ...", name)

	var dbName string
	var dbVersion int
	err := db.QueryRowContext(
		ctx,
		`SELECT name, version FROM lambda_function WHERE name = ? ORDER BY version DESC LIMIT 1`,
		name,
	).Scan(
		&dbName,
		&dbVersion,
	)

	switch {
	case err == sql.ErrNoRows:
		log.Info("... not found, returning version = 0.")
		return 0, nil
	case err != nil:
		msg := log.Error("error when querying function version for %s: %v", name, err)
		return -1, errors.New(msg)
	}

	log.Info("... found version %d.", dbVersion)

	return dbVersion, nil
}

func InsertFunction(ctx context.Context, db *database.Database, function *types.Function) (*types.Function, error) {

	tx, err := db.BeginTx(ctx)
	if err != nil {
		msg := log.Error("unable to create transaction to insert layer %s: %v",
			function.FunctionName, err)
		return nil, errors.New(msg)
	}

	functionId, err := tx.InsertOne(
		ctx,
		`INSERT INTO lambda_function (name, version, description, handler, role, dead_letter_arn,
					memory_size, runtime, timeout, code_sha256, code_size, last_modified_on)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`,
		function.FunctionName,
		function.Version,
		function.Description,
		function.Handler,
		function.Role,
		function.DeadLetterArn,
		function.MemorySize,
		function.Runtime,
		function.Timeout,
		function.CodeSha256,
		function.CodeSize,
		function.LastModified,
	)

	if err != nil {
		msg := tx.Rollback("unable to insert function %s", function.FunctionName)
		return nil, errors.New(msg)
	}

	tagsStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO lambda_function_tag (function_id, key, value) VALUES (?, ?, ?)`,
	)
	defer tagsStmt.Close()

	for key, value := range function.Tags {
		_, err := tagsStmt.ExecContext(ctx, functionId, key, value)
		if err != nil {
			msg := tx.Rollback("unable to add tag %s to function %s: %v", key, err)
			log.Error(msg)
			return nil, errors.New(msg)
		}
	}

	evnStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO lambda_function_environment (function_id, key, value) VALUES (?, ?, ?)`,
	)
	defer evnStmt.Close()

	for key, value := range function.Environment.Variables {
		_, err := evnStmt.ExecContext(ctx, functionId, key, value)
		if err != nil {
			msg := tx.Rollback("unable to add environment %s to function %s: %v", key, err)
			log.Error(msg)
			return nil, errors.New(msg)
		}
	}

	err = tx.Commit()
	if err != nil {
		msg := log.Error("unable to commit when inserting function %s: %v", function.FunctionName, err)
		return nil, errors.New(msg)
	}

	saved := types.Function{
		ID:                         functionId,
		FunctionName:               function.FunctionName,
		Description:                function.Description,
		Handler:                    function.Handler,
		Role:                       function.Role,
		DeadLetterArn:              function.DeadLetterArn,
		MemorySize:                 function.MemorySize,
		Runtime:                    function.Runtime,
		Timeout:                    function.Timeout,
		CodeSha256:                 function.CodeSha256,
		CodeSize:                   function.CodeSize,
		Environment:                function.Environment,
		Tags:                       function.Tags,
		LastModified:               function.LastModified,
		LastUpdateStatus:           "",
		LastUpdateStatusReason:     nil,
		LastUpdateStatusReasonCode: "",
		Layers:                     nil,
		PackageType:                "",
		RevisionId:                 nil,
		State:                      "",
		StateReason:                nil,
		StateReasonCode:            "",
		Version:                    function.Version,
	}

	return &saved, nil
}

func FunctionByName(ctx context.Context, db *database.Database, name string) (*types.Function, error) {
	var function types.Function
	err := db.QueryRowContext(
		ctx,
		`SELECT id, name, version, description, handler, role, dead_letter_arn, memory_size,
					runtime, timeout, code_sha256, code_size, last_modified_on
				FROM lambda_function WHERE name = ? ORDER BY version DESC LIMIT 1`,
		name,
	).Scan(
		&function.ID,
		&function.FunctionName,
		&function.Version,
		&function.Description,
		&function.Handler,
		&function.Role,
		&function.DeadLetterArn,
		&function.MemorySize,
		&function.Runtime,
		&function.Timeout,
		&function.CodeSha256,
		&function.CodeSize,
		&function.LastModified,
	)

	switch {
	case err == sql.ErrNoRows:
		log.Info("Querying function %s returned 0 rows.", name)
		return nil, err
	case err != nil:
		msg := log.Error("Error when querying for Function %s: %v", name, err)
		return nil, errors.New(msg)
	}

	log.Info("found function: %+v", function)
	log.Info("setting function %s version to $LATEST", name)

	function.Version = "$LATEST"

	return &function, nil
}
