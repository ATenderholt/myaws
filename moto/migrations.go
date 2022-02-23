package moto

import "myaws/database"

var Migrations = []database.Migration{
	{
		Service:     "Moto",
		Description: "Create Requests Table",
		Query: `CREATE TABLE IF NOT EXISTS moto_request (
					id             integer primary key autoincrement,
					service        text not null,
				    authorization  text not null,
                    content_type   text not null,
					payload        text not null
				);
		`,
	},
}