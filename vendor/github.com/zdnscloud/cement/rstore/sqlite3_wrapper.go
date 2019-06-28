package rstore

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
)

var sqlite3TypeMap = map[Datatype]string{
	String: "text",
	Int:    "integer",
	Uint32: "integer",
	Time:   "datetime",
	Bool:   "boolean",
}

func OpenSqlite3(path string) (*db, error) {
	conn, err := sql.Open("sqlite3", path+"?_foreign_keys=1")
	if err != nil {
		return nil, err
	} else {
		conn.SetMaxOpenConns(1)
		return &db{
			conn: conn,
			typ:  Sqlite3,
		}, nil
	}
}
