package rstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/lib/pq"
	"reflect"
	"strconv"
	"strings"
)

type DBType string

const (
	Postgresql DBType = "postgresql"
	Sqlite3    DBType = "sqlite3"
)

type db struct {
	conn *sql.DB
	typ  DBType
}

const (
	DefaultErrorMsg           = "inner db error"
	InvalidUpdateErrorMsg     = "invalid update"
	InvalidQueryErrorMsg      = "invalid query"
	DuplicateErrorMsg         = "duplicate resource"
	RelatedNoneExistsErrorMsg = "resource refer to doesn't exists"
)

var typeMap map[Datatype]string

func OpenDB(typ DBType, conf map[string]interface{}) (*db, error) {
	switch typ {
	case Postgresql:
		typeMap = postgresqlTypeMap
		return OpenPostgresql(conf["host"].(string),
			conf["user"].(string),
			conf["password"].(string),
			conf["dbname"].(string))
	case Sqlite3:
		typeMap = sqlite3TypeMap
		return OpenSqlite3(conf["path"].(string))
	default:
		panic("unknown db type")
	}
}

func (db *db) CloseDB() {
	db.conn.Close()
}

func (db *db) Exec(sql string) {
	db.conn.Exec(sql)
}

func (db *db) DropTable(tname string) {
	if db.typ == Sqlite3 {
		db.Exec("DROP TABLE IF EXISTS " + tname)
	} else {
		db.Exec("DROP TABLE IF EXISTS " + tname + " CASCADE")
	}
}

func (db *db) Begin() (*Tx, error) {
	tx, err := db.conn.Begin()
	if err == nil {
		return &Tx{tx}, nil
	} else {
		return nil, err
	}
}

func (db *db) BeginTx(level sql.IsolationLevel) (*Tx, error) {
	ctx := context.Background()
	tx, err := db.conn.BeginTx(ctx, &sql.TxOptions{
		Isolation: level,
	})

	if err == nil {
		return &Tx{tx}, nil
	} else {
		return nil, err
	}
}

func (db *db) HasTable(tname string) bool {
	_, err := db.conn.Query("SELECT * from " + tname + " limit 1")
	return err == nil
}

func convertArrayValue(array []interface{}) ([]interface{}, error) {
	result := make([]interface{}, 0, len(array))
	for _, elem := range array {
		kind := reflect.TypeOf(elem).Kind()
		if kind != reflect.Slice {
			result = append(result, elem)
		} else {
			var array_str []string
			if data, ok := elem.([]string); ok {
				for _, value := range data {
					array_str = append(array_str, ("\"" + value + "\""))
				}
				result = append(result, "{"+strings.Join(array_str, ",")+"}")
			} else if data, ok := elem.([]int); ok {
				for _, value := range data {
					array_str = append(array_str, strconv.Itoa(value))
				}
				result = append(result, "{"+strings.Join(array_str, ",")+"}")
			} else {
				return nil, errors.New("resource only support array of string and int")
			}
		}
	}
	return result, nil
}

func (db *db) PrepareAndExec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := db.conn.Prepare(query + ";")
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	validArgs, err := convertArrayValue(args)
	if err != nil {
		return nil, err
	}

	result, err := stmt.Exec(validArgs...)
	if err != nil {
		return nil, err
	}
	return result, err
}

type row []interface{}

type Tx struct {
	*sql.Tx
}

func (tx *Tx) DropTable(tname string) {
	tx.Exec("DROP TABLE IF EXISTS " + tname)
}

func (tx *Tx) HasTable(tname string) bool {
	_, err := tx.Query("SELECT * from " + tname + " limit 1")
	return err == nil
}

func (tx *Tx) PrepareAndExec(query string, args ...interface{}) (sql.Result, error) {
	stmt, err := tx.Prepare(query + ";")
	if err != nil {
		return nil, fmt.Errorf("%s:%s[%s]", InvalidUpdateErrorMsg, err.Error(), query)
	}
	defer stmt.Close()
	validArgs, err := convertArrayValue(args)
	if err != nil {
		return nil, fmt.Errorf("%s:%s[%s]", InvalidUpdateErrorMsg, err.Error(), query)
	}

	result, err := stmt.Exec(validArgs...)
	if err != nil {
		if strings.Contains(err.Error(), "violates unique") {
			return nil, fmt.Errorf("%s:%s[%s]", DuplicateErrorMsg, err.Error(), query)
		} else if strings.Contains(err.Error(), "violates foreign") {
			return nil, fmt.Errorf("%s:%s[%s]", RelatedNoneExistsErrorMsg, err.Error(), query)
		} else {
			return nil, fmt.Errorf("%s:%s[%s]", DefaultErrorMsg, err.Error(), query)
		}
	}
	return result, nil
}

func (tx *Tx) PrepareAndQuery(query string, args ...interface{}) ([]string, []row, error) {
	stmt, err := tx.Prepare(query + ";")
	if err != nil {
		return nil, nil, fmt.Errorf("%s:%s[%s]", InvalidQueryErrorMsg, err.Error(), query)
	}
	defer stmt.Close()
	validArgs, err := convertArrayValue(args)
	if err != nil {
		return nil, nil, fmt.Errorf("%s:%s[%s]", InvalidQueryErrorMsg, err.Error(), query)
	}

	rows, err := stmt.Query(validArgs...)
	if err != nil {
		return nil, nil, fmt.Errorf("%s:%s[%s]", InvalidQueryErrorMsg, err.Error(), query)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}

	out := []row{}
	for rows.Next() {
		r := make(row, 0, len(cols))
		for i := 0; i < cap(r); i++ {
			var v interface{}
			r = append(r, &v)
		}
		err := rows.Scan(r...)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, r)
	}
	return cols, out, nil
}
