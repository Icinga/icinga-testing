package utils

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strconv"
)

// DB is an interface that contains functions provided by *database/sql.DB and *github.com/jmoiron/sqlx.DB required by
// functions in this package so that in can be used with both.
type DB interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// PrettySelect performs a query on the given database connection, retrieves the result set, pretty-prints it and
// returns the result as a string suitable for writing into a log for debugging.
//
// Example usage:
//
//	t.Log(utils.MustT(t).String(utils.PrettySelect(db, "SELECT * FROM somewhere")))
func PrettySelect(db DB, query string, args ...interface{}) (string, error) {
	cursor, err := db.Query(query, args...)
	if err != nil {
		return "", fmt.Errorf("failed to execute query: %w", err)
	}
	defer func() { _ = cursor.Close() }()

	cols, err := cursor.Columns()
	if err != nil {
		return "", fmt.Errorf("failed to get columns of result: %w", err)
	}
	labelWidth := 0
	for _, col := range cols {
		l := len(col)
		if l > labelWidth {
			labelWidth = l
		}
	}

	var buf bytes.Buffer
	_, _ = fmt.Fprintf(&buf, "Query: %s\nArguments: %q\n", query, args)

	for rowId := 0; cursor.Next(); rowId++ {
		rawRow := make([]interface{}, len(cols))
		scanRow := make([]interface{}, len(cols))

		for i := range cols {
			scanRow[i] = &rawRow[i]
		}

		err := cursor.Scan(scanRow...)
		if err != nil {
			return "", fmt.Errorf("failed to scan row of result: %w", err)
		}

		_, _ = fmt.Fprintln(&buf)
		for i := range cols {
			_, _ = fmt.Fprintf(&buf, "[%d] %"+strconv.Itoa(labelWidth)+"s: %s\n",
				rowId, cols[i], prettyFormatValue(rawRow[i]))
		}
	}

	return buf.String(), nil
}

// prettyFormatValue pretty-prints a value received from the database.
func prettyFormatValue(value interface{}) string {
	switch v := value.(type) {
	case []byte:
		h := "0x" + hex.EncodeToString(v)
		s := fmt.Sprintf("%q", v)
		if len(h) < len(s) {
			return h
		} else {
			return s
		}
	case string:
		return fmt.Sprintf("%q", v)
	default:
		return fmt.Sprint(v)
	}
}
