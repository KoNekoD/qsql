package qsql

import (
	"database/sql"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pkg/errors"
	"reflect"
)

func scanAll(
	rows pgx.Rows,
	columnsNum int,
	positionsList [][][]int,
	dest []interface{},
) error {
	for rows.Next() {
		err := scanStructs(rows, columnsNum, positionsList, dest)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(rows.Err())
}

func scanFirst(
	rows pgx.Rows,
	columnsNum int,
	positionsList [][][]int,
	dest []interface{},
) error {
	if rows.Next() {
		err := scanStructs(rows, columnsNum, positionsList, dest)
		if err != nil {
			return err
		}

		return nil
	}

	if err := rows.Err(); err != nil {
		return errors.WithStack(err)
	}

	return sql.ErrNoRows
}

func scanStructs(
	rows pgx.Rows,
	lenColumns int,
	positionsList [][][]int,
	dest []interface{},
) error {
	allValues := make([]interface{}, 0, lenColumns)
	items := make([]reflect.Value, 0, len(dest))

	for i, dest := range dest {
		values, item := structValues(positionsList[i], dest)
		allValues = append(allValues, values...)
		items = append(items, item)
	}

	err := rows.Scan(allValues...)
	if err != nil {
		return errors.WithStack(err)
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

func structValues(
	positions [][]int,
	dest interface{},
) ([]interface{}, reflect.Value) {
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

func getPositionsList(
	columns []pgconn.FieldDescription,
	dest []interface{},
) ([][][]int, error) {
	positionsList := make([][][]int, 0, len(dest))

	var missedColumns []pgconn.FieldDescription

	for _, d := range dest {
		destType := reflect.TypeOf(d)
		if destType.Kind() != reflect.Pointer {
			return nil, fmt.Errorf("destination must be a pointer")
		}

		var structType reflect.Type

		sliceType := destType.Elem()
		switch { // struct or slice
		case sliceType.Kind() == reflect.Slice:
			structType = sliceType.Elem()
			if structType.Kind() == reflect.Pointer {
				structType = structType.Elem()
			}
		case sliceType.Kind() == reflect.Struct:
			structType = sliceType
		default:
			return nil, fmt.Errorf("destination must be a pointer of slice")
		}

		if structType.Kind() != reflect.Struct {
			return nil, fmt.Errorf("destination must be a pointer of slice of struct or pointer struct")
		}

		positions := getPositions(columns, structType)
		positionsList = append(positionsList, positions)
		for i, position := range positions {
			if position == nil {
				missedColumns = append(missedColumns, columns[i])
			}
		}
	}

	if len(missedColumns) != 0 {
		return nil, fmt.Errorf(
			"%d columns not found in destination structs",
			len(missedColumns),
		)
	}

	return positionsList, nil
}

func getPositions(columns []pgconn.FieldDescription, t reflect.Type) [][]int {
	positions := make([][]int, 0, t.NumField())
	discoverStruct(columns, &positions, t, nil)

	return positions
}

func discoverStruct(
	allColumns []pgconn.FieldDescription,
	positions *[][]int,
	t reflect.Type,
	prefix []int,
) int {
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
			used := discoverStruct(
				columns,
				positions,
				f.Type,
				append(prefix, i),
			)
			columns = columns[used:]

			continue
		}

		dbFieldsMap[name] = append(prefix, i)
	}

	for i := 0; i < len(dbFieldsMap); i++ {
		*positions = append(*positions, dbFieldsMap[columns[i].Name])
	}

	return len(dbFieldsMap)
}
