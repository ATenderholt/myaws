package lambda

import "myaws/database"

var Migrations = []database.Migration{
	{
		Service:     "Lambda",
		Description: "Create Layer Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_layer (
					id           integer primary key autoincrement,
					name         text not null,
					description  text not null,
					version      integer not null,
					created_on   integer not null,
					code_size	 integer not null,
					code_sha256  text not null
				);
		`,
	},
	{
		Service:     "Lambda",
		Description: "Create Runtime Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_runtime (
					id      integer primary key autoincrement,
					name	text not null unique
				);
			
				INSERT OR IGNORE INTO lambda_runtime (name) VALUES
				('python3.6'),
				('python3.7'),
				('python3.8');
		`,
	},
	{
		Service:     "Lambda",
		Description: "Create Layer Runtime Table",
		Query: `CREATE TABLE IF NOT EXISTS lambda_layer_runtime (
					id					integer primary key autoincrement,
					lambda_layer_id		integer,
					lambda_runtime_id	integer,
					FOREIGN KEY(lambda_layer_id) REFERENCES lambda_layer(id),
					FOREIGN	KEY(lambda_runtime_id) REFERENCES lambda_runtime(id)
				);
		`,
	},
	{
		Service:     "Lambda",
		Description: "Create Function & supporting Tables",
		Query: `CREATE TABLE IF NOT EXISTS lambda_function (
					id					integer primary key autoincrement,
					name				text not null,
					version				integer not null,
					description			text,
					handler				text not null,
					role				text,
					dead_letter_arn		text,
					memory_size			integer not null,
					runtime				text not null,
					timeout				integer not null,
					code_sha256			text not null,
					code_size			integer not null,
					last_modified_on	integer not null
				);

				CREATE TABLE IF NOT EXISTS lambda_function_environment (
					id					integer primary key autoincrement,
					function_id 		integer not null,
					key					text not null,
					value				text,
					FOREIGN KEY(function_id) REFERENCES lambda_function(id)
				);

				CREATE TABLE IF NOT EXISTS lambda_function_tag (
					id					integer primary key autoincrement,
					function_id 		integer not null,
					key					text not null,
					value				text,
					FOREIGN KEY(function_id) REFERENCES lambda_function(id)
				);

				CREATE TABLE IF NOT EXISTS lambda_function_layer (
					id					integer primary key autoincrement,
					function_id 		integer not null,
					layer_id			integer not null,
					FOREIGN KEY(function_id) REFERENCES lambda_function(id),
					FOREIGN KEY(layer_id) REFERENCES lambda_layer(id)
				);
		`,
	},
}
