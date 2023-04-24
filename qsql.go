package qsql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
)

type DB struct {
	db *sql.DB
}

func (db *DB) Select(sql string, args ...interface{}) *Query {
	return &Query{
		db:   db.db,
		sql:  sql,
		args: args,
	}
}

func (db *DB) Get(sql string, args ...interface{}) *Query {
	return &Query{
		db:        db.db,
		sql:       sql,
		args:      args,
		scanFirst: true,
	}
}

func (db *DB) Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	return db.db.ExecContext(ctx, sql, args...)
}

func (q *DB) Begin(ctx context.Context) (*TX, error) {
	tx, err := q.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return nil, err
	}

	return &TX{
		tx: tx,
	}, nil
}

type TX struct {
	tx *sql.Tx
}

func (tx *TX) Commit() error {
	return tx.tx.Commit()
}

func (tx *TX) Rollback() error {
	return tx.tx.Rollback()
}

func (tx *TX) Select(sql string, args ...interface{}) *Query {
	return &Query{
		db:   tx.tx,
		sql:  sql,
		args: args,
	}
}

func (tx *TX) Get(sql string, args ...interface{}) *Query {
	return &Query{
		db:        tx.tx,
		sql:       sql,
		args:      args,
		scanFirst: true,
	}
}

func (tx *TX) Exec(ctx context.Context, sql string, args ...interface{}) (sql.Result, error) {
	return tx.tx.ExecContext(ctx, sql, args...)
}

// type QueryBuilder struct {
// 	selects []string
// 	table   string
// 	limit   int
// 	offset  int
// }

// func (q *QueryBuilder) Select(f ...string) *QueryBuilder {
// 	q.selects = append(q.selects, f...)
// 	return q
// }

// func (q *QueryBuilder) From(table string) *QueryBuilder {
// 	q.table = table
// 	return q
// }

func Open(driverName, dataSourceName string) (*DB, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		db.Close()
		return nil, err
	}

	return &DB{
		db: db,
	}, nil
}

type Query struct {
	db interface {
		QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
	}

	sql       string
	args      []interface{}
	scan      []interface{}
	scanFirst bool

	postAction func(*sql.Rows, error)
}

func (q *Query) SetPostAction(postAction func(*sql.Rows, error)) {
	q.postAction = postAction
}

func (q *Query) Scan(dest ...interface{}) *Query {
	q.scan = append(q.scan, dest...)
	return q
}

func (q *Query) Exec(ctx context.Context) error {
	var (
		rows *sql.Rows
		err  error
	)

	if q.postAction != nil {
		defer func() {
			q.postAction(rows, err)
		}()
	}

	rows, err = q.db.QueryContext(ctx, q.sql, q.args...)
	if err != nil {
		return err
	}

	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	positionsList, err := getPositionsList(columns, q.scan)
	if err != nil {
		return err
	}

	if q.scanFirst {
		return scanFirst(rows, len(columns), positionsList, q.scan)
	}

	return scanAll(rows, len(columns), positionsList, q.scan)
}

func scanAll(rows *sql.Rows, columnsNum int, positionsList [][][]int, dest []interface{}) error {
	for rows.Next() {
		err := scanStructs(rows, columnsNum, positionsList, dest)
		if err != nil {
			return err
		}
	}

	return rows.Err()
}

func scanFirst(rows *sql.Rows, columnsNum int, positionsList [][][]int, dest []interface{}) error {
	if rows.Next() {
		err := scanStructs(rows, columnsNum, positionsList, dest)
		if err != nil {
			return err
		}

		return nil
	}

	if err := rows.Err(); err != nil {
		return err
	}

	return sql.ErrNoRows
}

func scanStructs(rows *sql.Rows, lenColumns int, positionsList [][][]int, dest []interface{}) error {
	allValues := make([]interface{}, 0, lenColumns)
	items := make([]reflect.Value, 0, len(dest))

	for i, dest := range dest {
		values, item := structValues(positionsList[i], dest)
		allValues = append(allValues, values...)
		items = append(items, item)
	}

	err := rows.Scan(allValues...)
	if err != nil {
		return err
	}

	for i, item := range items {
		if reflect.ValueOf(dest[i]).Elem().Kind() != reflect.Slice { // struct or slice
			reflect.ValueOf(dest[i]).Elem().Set(item.Elem())
			continue
		}

		slice := reflect.ValueOf(dest[i]).Elem()

		if slice.Type().Elem().Kind() == reflect.Pointer {
			slice.Set(reflect.Append(slice, item))
		} else {
			slice.Set(reflect.Append(slice, item.Elem()))
		}
	}

	return nil
}

func structValues(positions [][]int, dest interface{}) ([]interface{}, reflect.Value) {
	var values []interface{}

	structType := reflect.TypeOf(dest).Elem()
	if structType.Kind() == reflect.Slice { // struct or slice
		structType = structType.Elem()
	}

	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}

	item := reflect.New(structType)

	for _, p := range positions {
		values = append(values, getField(item.Elem(), p).Addr().Interface())
	}

	return values, item
}

func getField(v reflect.Value, p []int) reflect.Value {
	if len(p) == 1 {
		return v.Field(p[0])
	}

	return getField(v.Field(p[0]), p[1:])
}

func getPositionsList(columns []string, dest []interface{}) ([][][]int, error) {
	positionsList := make([][][]int, 0, len(dest))

	for _, d := range dest {
		destType := reflect.TypeOf(d)
		if destType.Kind() != reflect.Pointer {
			return nil, fmt.Errorf("destination must be a pointer")
		}

		var structType reflect.Type

		sliceType := destType.Elem()

		if sliceType.Kind() == reflect.Slice { // struct or slice
			structType = sliceType.Elem()
			if structType.Kind() == reflect.Pointer {
				structType = structType.Elem()
			}
		} else if sliceType.Kind() == reflect.Struct {
			structType = sliceType
		} else {
			return nil, fmt.Errorf("destination must be a pointer of slice")
		}

		if structType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("destination must be a pointer of slice of struct or pointer struct")
		}

		positions := getPositions(columns, structType)
		positionsList = append(positionsList, positions)
		columns = columns[len(positions):]
	}

	if len(columns) != 0 {
		return nil, fmt.Errorf("%d columns not found in destination structs", len(columns))
	}

	return positionsList, nil
}

func getPositions(columns []string, t reflect.Type) [][]int {
	positions := make([][]int, 0, t.NumField())
	discoverStruct(columns, &positions, t, nil)
	return positions
}

func discoverStruct(allColumns []string, positions *[][]int, t reflect.Type, prefix []int) int {
	dbFieldsMap := make(map[string][]int, t.NumField())
	columns := allColumns

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)

		name := f.Tag.Get("db")

		if !f.IsExported() || name == "-" {
			continue
		}

		if name == "" {
			name = f.Name
		}

		if f.Type.Kind() == reflect.Struct && (f.Anonymous || name == "*") {
			columns = columns[len(dbFieldsMap):]
			used := discoverStruct(columns, positions, f.Type, append(prefix, i))
			columns = columns[used:]
			continue
		}

		dbFieldsMap[name] = append(prefix, i)
	}

	for i := 0; i < len(dbFieldsMap); i++ {
		*positions = append(*positions, dbFieldsMap[columns[i]])
	}

	return len(dbFieldsMap)
}
