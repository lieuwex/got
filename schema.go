package main

import (
	"database/sql"
	"fmt"
)

var schemaEntries = []string{
	`CREATE TABLE entries (id integer NOT NULL PRIMARY KEY AUTOINCREMENT, note varchar(255), start timestamp, end timestamp, sheet varchar(255));`,
	`CREATE TABLE meta (id integer NOT NULL PRIMARY KEY AUTOINCREMENT, key varchar(255), value varchar(255));`,

	`insert into meta(key, value) values("last_checkout_id", 0)`,
	`insert into meta(key, value) values("current_sheet", "main")`,
	`insert into meta(key, value) values("last_sheet", "main")`,
}

func runSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, entry := range schemaEntries {
		fmt.Printf("running: %s\n", entry)
		if _, err := tx.Exec(entry); err != nil {
			return err
		}
	}

	return tx.Commit()
}
