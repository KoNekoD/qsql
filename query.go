package qsql

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
)

type Query struct {
	db *pgxpool.Pool

	sql       string
	args      []interface{}
	scan      []interface{}
	scanFirst bool
}

func (q *Query) Scan(dest ...interface{}) *Query {
	q.scan = append(q.scan, dest...)

	return q
}

func (q *Query) Exec(ctx context.Context) error {
	rows, err := q.db.Query(ctx, q.sql, q.args...)
	if err != nil {
		return errors.WithStack(err)
	}

	defer rows.Close()

	columns := rows.FieldDescriptions()

	positionsList, err := getPositionsList(columns, q.scan)
	if err != nil {
		return err
	}

	if q.scanFirst {
		return scanFirst(rows, len(columns), positionsList, q.scan)
	}

	return scanAll(rows, len(columns), positionsList, q.scan)
}
