package rstore

import (
	"database/sql"
	"fmt"
	"reflect"
	"text/template"

	"cement/reflector"
)

type RStore struct {
	db   *db
	meta *ResourceMeta
}

type RStoreTx struct {
	tx   *Tx
	meta *ResourceMeta
}

const (
	joinSqlTemplateContent string = "select {{.OwnedTable}}.* from {{.OwnedTable}} inner join {{.RelTable}} on ({{.OwnedTable}}.id={{.RelTable}}.{{.Owned}} and {{.RelTable}}.{{.Owner}}=$1)"
)

var joinSqlTemplate *template.Template

func init() {
	joinSqlTemplate, _ = template.New("").Parse(joinSqlTemplateContent)
}

func NewRStore(typ DBType, connInfo map[string]interface{}, meta *ResourceMeta) (ResourceStore, error) {
	db, err := OpenDB(typ, connInfo)
	if err != nil {
		return nil, err
	}

	for _, descriptor := range meta.GetDescriptors() {
		db.Exec(createTableSql(descriptor))
	}

	return &RStore{db, meta}, nil
}

func (store *RStore) Destroy() {
	store.db.CloseDB()
}

func (store *RStore) Clean() {
	resources := store.meta.Resources()
	for i := len(resources); i > 0; i-- {
		store.db.DropTable(resourceTableName(resources[i-1]))
	}
}

func (store *RStore) Begin() (Transaction, error) {
	tx, err := store.db.Begin()
	if err != nil {
		return nil, err
	} else {
		return &RStoreTx{tx, store.meta}, nil
	}
}

func (store *RStore) BeginTx(level sql.IsolationLevel) (Transaction, error) {
	tx, err := store.db.BeginTx(level)
	if err != nil {
		return nil, err
	} else {
		return &RStoreTx{tx, store.meta}, nil
	}
}

//note, if there is any error occured in tx
//if db is postgresql commit will fail, rollback is automatically called
//if db is sqlite3 rollback must be called mannually, otherwise operation
//which succeed in this tx will be commit
func (tx *RStoreTx) RollBack() error {
	return tx.tx.Rollback()
}

func (tx *RStoreTx) Commit() error {
	return tx.tx.Commit()
}

func (tx *RStoreTx) Insert(r Resource) (Resource, error) {
	//this may change the id of r, if id isn't specified
	sql, args, err := insertSqlArgsAndID(tx.meta, r)
	if err != nil {
		return nil, err
	}

	_, err = tx.tx.PrepareAndExec(sql, args...)
	if err != nil {
		return nil, err
	} else {
		return r, err
	}
}

func (tx *RStoreTx) GetOwned(owner ResourceType, ownerID string, owned ResourceType) (interface{}, error) {
	rt, err := tx.meta.NewDefault(owned)
	if err != nil {
		return nil, err
	}

	slicePointer := reflector.NewSlicePointer(reflect.ValueOf(rt).Elem().Type())
	sql, args, err := joinSelectSqlAndArgs(tx.meta, owner, owned, ownerID)
	if err != nil {
		return nil, err
	}

	err = tx.getWithSql(sql, args, slicePointer)
	if err != nil {
		return nil, err
	} else {
		return reflect.ValueOf(slicePointer).Elem().Interface(), nil
	}
}

func (tx *RStoreTx) FillOwned(owner ResourceType, ownerID string, out interface{}) error {
	pr, err := reflector.GetStructInSlice(out)
	if err != nil {
		return err
	}

	r, _ := pr.(Resource)
	sql, args, err := joinSelectSqlAndArgs(tx.meta, owner, GetResourceType(r), ownerID)
	if err != nil {
		return err
	}

	return tx.getWithSql(sql, args, out)
}

func (tx *RStoreTx) Get(typ ResourceType, conds map[string]interface{}) (interface{}, error) {
	rt, err := tx.meta.NewDefault(typ)
	if err != nil {
		return nil, err
	}

	p := reflector.NewSlicePointer(reflect.ValueOf(rt).Elem().Type())
	err = tx.Fill(conds, p)
	if err != nil {
		return nil, err
	} else {
		return reflect.ValueOf(p).Elem().Interface(), nil
	}
}

func (tx *RStoreTx) Fill(conds map[string]interface{}, out interface{}) error {
	pr, err := reflector.GetStructInSlice(out)
	if err != nil {
		return err
	}

	r, _ := pr.(Resource)
	sql, args, err := selectSqlAndArgs(tx.meta, GetResourceType(r), conds)
	if err != nil {
		return err
	}
	return tx.getWithSql(sql, args, out)
}

func (tx *RStoreTx) getWithSql(sql string, args []interface{}, out interface{}) error {
	slice := reflect.Indirect(reflect.ValueOf(out))
	r := slice.Type().Elem()
	cols, rows, err := tx.tx.PrepareAndQuery(sql, args...)
	if err != nil {
		return err
	}

	for _, row := range rows {
		r_, err := rowToStruct(cols, row, r)
		if err != nil {
			return err
		}
		slice.Set(reflect.Append(slice, (*r_).Elem()))
	}
	return nil
}

func (tx *RStoreTx) Exists(typ ResourceType, conds map[string]interface{}) (bool, error) {
	sql, params, err := existsSqlAndArgs(tx.meta, typ, conds)
	if err != nil {
		return false, err
	}

	return tx.existsWithSql(sql, params...)
}

func (tx *RStoreTx) existsWithSql(sql string, params ...interface{}) (bool, error) {
	_, rows, err := tx.tx.PrepareAndQuery(sql, params...)

	if err == nil && len(rows) == 1 {
		v := reflect.Indirect(reflect.ValueOf(rows[0][0]))
		if v.Elem().Kind() == reflect.Int64 {
			return v.Elem().Int() != 0, nil
		} else {
			return v.Elem().Bool(), nil
		}
	} else {
		return false, err
	}
}

func (tx *RStoreTx) Count(typ ResourceType, conds map[string]interface{}) (int64, error) {
	sql, params, err := countSqlAndArgs(tx.meta, typ, conds)
	if err != nil {
		return 0, err
	}

	return tx.countWithSql(sql, params...)
}

func (tx *RStoreTx) CountEx(typ ResourceType, sql string, params ...interface{}) (int64, error) {
	if tx.meta.Has(typ) == false {
		return 0, fmt.Errorf("unknown resource type %v", typ)
	}
	return tx.countWithSql(sql, params...)
}

func (tx *RStoreTx) countWithSql(sql string, params ...interface{}) (int64, error) {
	_, rows, err := tx.tx.PrepareAndQuery(sql, params...)

	if err == nil && len(rows) == 1 {
		v := reflect.Indirect(reflect.ValueOf(rows[0][0]))
		return int64(v.Elem().Int()), nil
	} else {
		return 0, err
	}
}

func (tx *RStoreTx) Update(typ ResourceType, nv map[string]interface{}, conds map[string]interface{}) (int64, error) {
	sql, args, err := updateSqlAndArgs(tx.meta, typ, nv, conds)
	if err != nil {
		return 0, err
	}

	result, err := tx.tx.PrepareAndExec(sql, args...)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}
}

func (tx *RStoreTx) Delete(typ ResourceType, conds map[string]interface{}) (int64, error) {
	sql, args, err := deleteSqlAndArgs(tx.meta, typ, conds)
	if err != nil {
		return 0, err
	}

	result, err := tx.tx.PrepareAndExec(sql, args...)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}
}

func (tx *RStoreTx) DeleteEx(sql string, params ...interface{}) (int64, error) {
	return tx.Exec(sql, params...)
}

func (tx *RStoreTx) GetEx(typ ResourceType, sql string, params ...interface{}) (interface{}, error) {
	rt, err := tx.meta.NewDefault(typ)
	if err != nil {
		return nil, err
	}

	p := reflector.NewSlicePointer(reflect.ValueOf(rt).Elem().Type())
	err = tx.FillEx(p, sql, params...)
	if err != nil {
		return nil, err
	} else {
		return reflect.ValueOf(p).Elem().Interface(), nil
	}
}

func (tx *RStoreTx) GetDefaultResource(typ ResourceType) (Resource, error) {
	return tx.meta.NewDefault(typ)
}

func (tx *RStoreTx) FillEx(out interface{}, sql string, params ...interface{}) error {
	return tx.getWithSql(sql, params, out)
}

func (tx *RStoreTx) Exec(sql string, params ...interface{}) (int64, error) {
	result, err := tx.tx.PrepareAndExec(sql, params...)
	if err != nil {
		return 0, err
	} else {
		return result.RowsAffected()
	}
}
