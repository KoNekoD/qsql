# qsql

[![Go Reference](https://pkg.go.dev/badge/github.com/dedalqq/qsql.svg)](https://pkg.go.dev/github.com/dedalqq/qsql)

qsql - is a simple library for scan selected database data in to golang structures

it's library was inspired sqlx, this library can scan into multiple structures

## Examples

Open connect to database and exec SQL query
```go
package main

import (
    "context"

    "github.com/dedalqq/qsql"
)

func main() {
    ctx := context.Background()

	db, err := qsql.Open("pgx", "postgres://<login>:<password>@<host>")
	if err != nil {
		panic(err)
	}

    sql := `SELECT * FROM users WHERE id = $1;`

    var user []User

    err = db.Select(sql, 123).Scan(&user).Exec(ctx)
    if err != nil {
		panic(err)
	}
}
```

Also you can do:

Select user with count:
```go
sql := `SELECT u.*, COUNT(*) OVER() AS count FROM users AS u LIMIT $1;`

var message []Message
var count struct{
    Count int `db:count`
}()

err := db.Select(sql, 10).Scan(&user, &count).Exec(ctx)
```

Notice: Structures Message and User may have the same fields, such as `id`, `status`, `created` and others

Select message with users:
```go
sql := `SELECT m.*, u.* FROM message AS m
    LEFT JOIN users AS u ON m.user_id = m.id;`

var message []Message
var user []User

err := db.Select(sql).Scan(&message, &user).Exec(ctx)
```

Or so:
```go
sql := `SELECT m.*, u.* FROM message AS m
    LEFT JOIN users AS u ON m.user_id = m.id;`

var data []struct{
    message Message
    user User
}{}

err := db.Select(sql).Scan(&data).Exec(ctx)
```

Select user and all his message:
```go
sql := `SELECT u.*, m.* FROM users AS u
    LEFT JOIN message AS m ON m.user_id = m.id
    WHERE id = $1;`

var user User
var message []Message

err := db.Select(sql, 123).Scan(&user, &message).Exec(ctx)
```
