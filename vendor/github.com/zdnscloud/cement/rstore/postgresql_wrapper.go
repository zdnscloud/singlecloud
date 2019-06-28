package rstore

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"strconv"
	"strings"
)

var postgresqlTypeMap = map[Datatype]string{
	String:      "text",
	Int:         "integer",
	Uint32:      "bigint",
	Time:        "timestamp with time zone",
	IntArray:    "integer[]",
	StringArray: "text[]",
	Bool:        "boolean",
}

func OpenPostgresql(host, user, passwd, dbname string) (*db, error) {
	port := 5432
	hostAndPort := strings.Split(host, ":")
	if len(hostAndPort) == 2 {
		host = hostAndPort[0]
		port, _ = strconv.Atoi(hostAndPort[1])
	} else {
		host = hostAndPort[0]
	}
	var conninfo = fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable",
		host,
		port,
		user,
		dbname,
		passwd,
	)
	conn, err := sql.Open("postgres", conninfo)
	if err != nil {
		return nil, err
	} else {
		return &db{
			conn: conn,
			typ:  Postgresql,
		}, nil
	}
}
