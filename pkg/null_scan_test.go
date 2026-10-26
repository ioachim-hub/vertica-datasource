package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	"google.golang.org/protobuf/proto"
)

// Exercise database/sql's real NULL conversions with deterministic driver rows.
func TestNullableScanToFrame(t *testing.T) {
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		dbType string
		value  driver.Value
		want   interface{}
	}{
		{"VARCHAR", "/aim/a/radcom-aim-app/tcf-rca?timeseries_uuid=418", "/aim/a/radcom-aim-app/tcf-rca?timeseries_uuid=418"},
		{"CHAR", "x", "x"}, {"LONG VARCHAR", "text", "text"}, {"UUID", "uuid", "uuid"},
		{"INTERVAL DAY TO SECOND", "1 day", "1 day"}, {"UNKNOWN", "text", "text"},
		{"BOOL", false, false}, {"INT", int64(0), int64(0)}, {"FLOAT", float64(0), float64(0)},
		{"NUMERIC", "12.5", float64(12.5)},
		{"DATE", now, now}, {"TIMESTAMP", now, now}, {"TIMESTAMPTZ", now, now},
		{"TIME", now, now}, {"TIMETZ", now, now},
		{"BINARY", []byte{0xff, 0}, "ff00"}, {"VARBINARY", []byte{0xff, 0}, "ff00"},
		{"LONG VARBINARY", []byte{0xff, 0}, "ff00"},
	} {
		t.Run(tc.dbType, func(t *testing.T) {
			values := []driver.Value{nil, tc.value, nil}
			wants := []interface{}{nil, tc.want, nil}
			if tc.dbType == "VARCHAR" {
				values = append(values, "", []byte{0xff, 'a'})
				wants = append(wants, "", "\uFFFDa")
			}
			if tc.dbType == "BINARY" || tc.dbType == "VARBINARY" || tc.dbType == "LONG VARBINARY" {
				values = append(values, []byte{})
				wants = append(wants, "")
			}
			db := sql.OpenDB(scanConnector{dbType: tc.dbType, values: values})
			defer db.Close()
			rows, err := db.QueryContext(context.Background(), "SELECT rca_url")
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()
			columns, err := rows.ColumnTypes()
			if err != nil {
				t.Fatal(err)
			}
			frame := data.NewFrameOfFieldTypes("A", 0, generateFrameType(columns)...)
			index := 0
			for rows.Next() {
				scanned := generateRowIn(columns)
				if err := rows.Scan(scanned...); err != nil {
					t.Fatalf("row %d: %v", index, err)
				}
				frame.AppendRow(prepareRowForFrame([]string{tc.dbType}, scanned)...)
				got := reflect.ValueOf(frame.Fields[0].At(index))
				var actual interface{}
				if !got.IsNil() {
					actual = got.Elem().Interface()
				}
				if !reflect.DeepEqual(actual, wants[index]) {
					t.Fatalf("row %d = %#v, want %#v", index, actual, wants[index])
				}
				index++
			}
			if err := rows.Err(); err != nil {
				t.Fatal(err)
			}
			if index != len(wants) {
				t.Fatalf("got %d rows, want %d", index, len(wants))
			}
			response := &backend.QueryDataResponse{Responses: backend.Responses{"A": {Frames: data.Frames{frame}}}}
			encoded, err := backend.ToProto().QueryDataResponse(backend.DataFrameFormat_ARROW, response)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := proto.Marshal(encoded); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type scanConnector struct {
	dbType string
	values []driver.Value
}

func (c scanConnector) Connect(context.Context) (driver.Conn, error) { return scanConn{c}, nil }
func (c scanConnector) Driver() driver.Driver                        { return scanDriver{} }

type scanDriver struct{}

func (scanDriver) Open(string) (driver.Conn, error) { return nil, fmt.Errorf("use connector") }

type scanConn struct{ scanConnector }

func (scanConn) Prepare(string) (driver.Stmt, error) { return nil, fmt.Errorf("not supported") }
func (scanConn) Close() error                        { return nil }
func (scanConn) Begin() (driver.Tx, error)           { return nil, fmt.Errorf("not supported") }
func (c scanConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &scanRows{dbType: c.dbType, values: c.values}, nil
}

type scanRows struct {
	dbType string
	values []driver.Value
	index  int
}

func (*scanRows) Columns() []string                       { return []string{"rca_url"} }
func (*scanRows) Close() error                            { return nil }
func (r *scanRows) ColumnTypeDatabaseTypeName(int) string { return r.dbType }
func (r *scanRows) Next(dest []driver.Value) error {
	if r.index == len(r.values) {
		return io.EOF
	}
	dest[0] = r.values[r.index]
	r.index++
	return nil
}
