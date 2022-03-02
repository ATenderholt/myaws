package queries

import (
	"context"
	"database/sql"
	"errors"
	aws "github.com/aws/aws-sdk-go-v2/service/lambda/types"
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

	layerStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO lambda_function_layer (function_id, layer_name, layer_version) VALUES (?, ?, ?)`,
	)
	if err != nil {
		msg := tx.Rollback("unable to create statement to add layers to function %s", function.FunctionName)
		log.Error(msg)
		return nil, errors.New(msg)
	}
	defer layerStmt.Close()

	for _, layer := range function.Layers {
		_, err := layerStmt.ExecContext(ctx, functionId, layer.Name, layer.Version)
		if err != nil {
			msg := tx.Rollback("unable to add layer %s to function %s: %v", layer.Name, function.FunctionName, err)
			log.Error(msg)
			return nil, errors.New(msg)
		}
	}

	tagsStmt, err := tx.PrepareContext(
		ctx,
		`INSERT INTO lambda_function_tag (function_id, key, value) VALUES (?, ?, ?)`,
	)
	defer tagsStmt.Close()

	for key, value := range function.Tags {
		_, err := tagsStmt.ExecContext(ctx, functionId, key, value)
		if err != nil {
			msg := tx.Rollback("unable to add tag %s to function %s: %v", key, function.FunctionName, err)
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

func LatestFunctionByName(ctx context.Context, db *database.Database, name string) (*types.Function, error) {
	log.Info("Querying for Latest Function %s ... ", name)

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
		log.Info("... found 0 rows for Function %s.", name)
		return nil, err
	case err != nil:
		msg := log.Error("... error when querying for Function %s: %v", name, err)
		return nil, errors.New(msg)
	}

	log.Info("Found Function: %+v", function)
	log.Info("Setting Function %s version to $LATEST", name)

	function.Version = "$LATEST"

	environment, err := GetEnvironmentForFunction(ctx, db, &function)
	if err != nil {
		return nil, err
	}

	function.Environment = environment

	return &function, nil
}

func GetEnvironmentForFunction(ctx context.Context, db *database.Database, function *types.Function) (*aws.Environment, error) {
	log.Info("Querying environment for Function %s ...", function.FunctionName)

	variables := make(map[string]string)
	results, err := db.QueryContext(
		ctx,
		`SELECT key, value FROM lambda_function_environment WHERE function_id=?`,
		function.ID,
	)
	switch {
	case err == sql.ErrNoRows:
		log.Info("No Environment was found for Function %s", function.FunctionName)
	case err != nil:
		msg := log.Error("Unable to get Environment for Function %s: %v", function.FunctionName, err)
		return nil, errors.New(msg)
	}

	for results.Next() {
		var key, value string
		err = results.Scan(&key, &value)
		if err != nil {
			msg := log.Error("Unable to Scan Environment for Function %s: %v", function.FunctionName, err)
			return nil, errors.New(msg)
		}

		variables[key] = value
	}

	return &aws.Environment{Variables: variables}, nil
}

func FunctionVersionsByName(ctx context.Context, db *database.Database, name string) ([]types.Function, error) {
	log.Info("Querying for all Versions of Function %s", name)

	var results []types.Function
	rows, err := db.QueryContext(
		ctx,
		`SELECT id, name, version, description, handler, role, dead_letter_arn, memory_size,
					runtime, timeout, code_sha256, code_size, last_modified_on
				FROM lambda_function WHERE name = ?`,
		name,
	)

	if err != nil {
		msg := log.Error("Error when querying for all versions of Function %s: %v", name, err)
		return nil, errors.New(msg)
	}

	for rows.Next() {
		var function types.Function
		err := rows.Scan(
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

		if err != nil {
			progress := len(results)
			msg := log.Error("Error when scanning row %d for Function %s: %v", progress, name, err)
			return nil, errors.New(msg)
		}

		environment, err := GetEnvironmentForFunction(ctx, db, &function)
		if err != nil {
			progress := len(results)
			msg := log.Error("Error when hydrating row %d for Function %s: %v", progress, name, err)
			return nil, errors.New(msg)
		}

		function.Environment = environment

		results = append(results, function)
	}

	return results, nil
}

func GetLayersForFunction(ctx context.Context, db *database.Database, function *types.Function) ([]types.LambdaLayer, error) {
	log.Info("Querying for Layers of Function %s ...", function.FunctionName)
	layers := []types.LambdaLayer{}

	rows, err := db.QueryContext(
		ctx,
		`SELECT layer_name, layer_version, code_size FROM lambda_function_layer AS lfl
					JOIN lambda_layer AS ll
						ON lfl.layer_name = ll.name AND lfl.layer_version = ll.version
				WHERE lfl.function_id = ?
		`,
		function.ID,
	)

	switch {
	case err == sql.ErrNoRows:
		log.Info("... no layers were found for Function %s.", function.FunctionName)
		return layers, nil
	case err != nil:
		msg := log.Error("Error when querying for Layers of Function %s: %v", function.FunctionName, err)
		return nil, errors.New(msg)
	}

	for rows.Next() {
		var layer types.LambdaLayer
		err := rows.Scan(&layer.Name, &layer.Version, &layer.CodeSize)
		if err != nil {
			msg := log.Error("Error when scanning Layer results for Function %s: %v", function.FunctionName, err)
			return layers, errors.New(msg)
		}

		layers = append(layers, layer)
	}

	log.Info("... Found %d Layers for Function %s.", len(layers), function.FunctionName)
	return layers, nil
}

func LatestFunctions(ctx context.Context, db *database.Database) ([]types.Function, error) {
	log.Info("Querying for latest version of all Functions ...")

	rows, err := db.QueryContext(
		ctx,
		`SELECT name, max(version), runtime, handler FROM lambda_function GROUP BY name`,
	)

	var results []types.Function

	switch {
	case err == sql.ErrNoRows:
		log.Error("No Functions were found")
		return results, nil
	case err != nil:
		msg := log.Error("Unable to find Functions: %v", err)
		return results, errors.New(msg)
	}

	for rows.Next() {
		var function types.Function

		err := rows.Scan(
			&function.FunctionName,
			&function.Version,
			&function.Runtime,
			&function.Handler,
		)

		if err != nil {
			msg := log.Error("Unable to scan Function row #%d", len(results))
			return results, errors.New(msg)
		}

		results = append(results, function)
	}

	log.Info("... found %d Functions.", len(results))
	return results, nil
}

func UpsertFunctionEnvironment(ctx context.Context, db *database.Database, function *types.Function, environment *aws.Environment) error {
	log.Info("Upserting Environment for Function %s ...", function.FunctionName)

	adds := make(map[string]string)
	removes := make(map[string]string)
	updates := make(map[string]string)

	for key, value := range environment.Variables {
		_, exists := function.Environment.Variables[key]
		if exists {
			updates[key] = value
		} else {
			adds[key] = value
		}
	}

	for key, value := range function.Environment.Variables {
		_, exists := environment.Variables[key]
		if !exists {
			removes[key] = value
		}
	}

	if len(adds) == 0 && len(removes) == 0 && len(updates) == 0 {
		log.Info("No changes to Environment for Function %s", function.FunctionName)
		return nil
	}

	log.Info(" ... %d adds, %d updates, %d removes ...", len(adds), len(updates), len(removes))

	tx, err := db.BeginTx(ctx)
	if err != nil {
		msg := log.Error("Unable to begin transaction to change Environment for Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	addStmt, updateStmt, removeStmt, err := prepareEnvironmentUpdateStatements(ctx, tx)
	if err != nil {
		msg := log.Error("Unable to create statements to change Environment for Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}
	defer addStmt.Close()
	defer updateStmt.Close()
	defer removeStmt.Close()

	for key, value := range adds {
		_, err = addStmt.ExecContext(ctx, function.ID, key, value)
		if err != nil {
			msg := tx.Rollback("Unable to add Environment %s=%s to Function %s: %v", key, value, function.ID, err)
			return errors.New(msg)
		}
	}

	for key, value := range updates {
		_, err = updateStmt.ExecContext(ctx, value, function.ID, key)
		if err != nil {
			msg := tx.Rollback("Unable to update Environment %s=%s for Function %s: %v", key, value, function.ID, err)
			return errors.New(msg)
		}
	}

	for key, value := range removes {
		_, err = removeStmt.ExecContext(ctx, function.ID, key)
		if err != nil {
			msg := tx.Rollback("Unable to update Environment %s=%s for Function %s: %v", key, value, function.ID, err)
			return errors.New(msg)
		}
	}

	err = tx.Commit()

	if err != nil {
		msg := log.Error("Unable to commit Environment changes for Function %s: %v", function.FunctionName, err)
		return errors.New(msg)
	}

	function.Environment = environment

	log.Info(" ... finished.")

	return nil
}

func prepareEnvironmentUpdateStatements(ctx context.Context, tx *database.Transaction) (add *sql.Stmt, update *sql.Stmt, remove *sql.Stmt, err error) {
	add, err = tx.PrepareContext(ctx, `INSERT INTO lambda_function_environment (function_id, key, value) VALUES (?, ?, ?)`)
	if err != nil {
		msg := log.Error("Unable to create add statement to change Environment: %v", err)
		err = errors.New(msg)
		return
	}

	update, err = tx.PrepareContext(ctx, `UPDATE lambda_function_environment SET value=? WHERE function_id=? AND key=?`)
	if err != nil {
		msg := log.Error("Unable to create update statement to change Environment: %v", err)
		err = errors.New(msg)
		return
	}

	remove, err = tx.PrepareContext(ctx, `DELETE FROM lambda_function_environment WHERE function_id=? AND key=?`)
	if err != nil {
		msg := log.Error("Unable to create remove statement to change Environment: %v", err)
		err = errors.New(msg)
		return
	}

	return
}
