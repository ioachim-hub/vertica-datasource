package main

import (
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

type datasourceConfig struct {
	Database                   string `json:"database"`
	Host                       string `json:"host"`
	UseConnectionLoadbalancing bool   `json:"useConnectionLoadbalancing"`
	UsePreparedStatement       bool   `json:"usePreparedStatement"`
	User                       string `json:"user"`
	SSlMode                    string `json:"sslMode,omitempty"`
	MaxOpenConnections         int    `json:"maxOpenConnections"`
	MaxIdealConnections        int    `json:"maxIdealConnections"`
	MaxConnectionIdealTime     int    `json:"maxConnectionIdealTime"`
}

func (config *datasourceConfig) ConnectionURL(password string) string {
	var tlsmode string
	if config.SSlMode == "" {
		tlsmode = "none"
	} else {
		tlsmode = config.SSlMode
	}

	return fmt.Sprintf("vertica://%s:%s@%s/%s?use_prepared_statements=%d&connection_load_balance=%d&tlsmode=%s",
		config.User, password, config.Host, config.Database, boolToUint8(config.UsePreparedStatement), boolToUint8(config.UseConnectionLoadbalancing), tlsmode)
}

func boolToUint8(x bool) int8 {
	var y int8
	if x {
		y = 1
	}
	return y
}

func generateFrameType(columnTypes []*sql.ColumnType) []data.FieldType {
	colTypes := make([]data.FieldType, 0)
	for _, col := range columnTypes {
		switch col.DatabaseTypeName() {
		case "BOOL":
			colTypes = append(colTypes, data.FieldTypeNullableBool)

		case "INT":
			colTypes = append(colTypes, data.FieldTypeNullableInt64)

		case "FLOAT":
			colTypes = append(colTypes, data.FieldTypeNullableFloat64)

		case "CHAR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "VARCHAR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "DATE":
			colTypes = append(colTypes, data.FieldTypeNullableTime)

		case "TIMESTAMP":
			colTypes = append(colTypes, data.FieldTypeNullableTime)

		case "TIMESTAMPTZ":
			colTypes = append(colTypes, data.FieldTypeNullableTime)

		case "INTERVAL DAY":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL DAY TO SECOND":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL HOUR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL HOUR TO MINUTE":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL HOUR TO SECOND":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL MINUTE":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL MINUTE TO SECOND":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL SECOND":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL DAY TO HOUR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL DAY TO MINUTE":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL YEAR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL YEAR TO MONTH":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "INTERVAL MONTH":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "TIME":
			colTypes = append(colTypes, data.FieldTypeNullableTime)

		case "TIMETZ":
			colTypes = append(colTypes, data.FieldTypeNullableTime)

		case "VARBINARY":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "UUID":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "LONG VARCHAR":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "LONG VARBINARY":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "BINARY":
			colTypes = append(colTypes, data.FieldTypeNullableString)

		case "NUMERIC":
			colTypes = append(colTypes, data.FieldTypeNullableFloat64)

		default:
			colTypes = append(colTypes, data.FieldTypeNullableString)
		}
	}
	return colTypes
}

func generateRowIn(columnTypes []*sql.ColumnType) []interface{} {
	rowIn := make([]interface{}, 0)
	for _, fieldType := range generateFrameType(columnTypes) {
		// database/sql accepts **T: SQL NULL leaves *T nil, other values allocate T.
		switch fieldType {
		case data.FieldTypeNullableBool:
			var i *bool
			rowIn = append(rowIn, &i)

		case data.FieldTypeNullableInt64:
			var i *int64
			rowIn = append(rowIn, &i)

		case data.FieldTypeNullableFloat64:
			var i *float64
			rowIn = append(rowIn, &i)

		case data.FieldTypeNullableString:
			var i *string
			rowIn = append(rowIn, &i)

		case data.FieldTypeNullableTime:
			var i *time.Time
			rowIn = append(rowIn, &i)
		}
	}
	// Binary fields are represented as strings in Grafana, but scanned as bytes.
	for i, col := range columnTypes {
		switch col.DatabaseTypeName() {
		case "VARBINARY", "LONG VARBINARY", "BINARY":
			var raw []byte
			rowIn[i] = &raw
		}
	}
	return rowIn
}

func prepareRowForFrame(databaseTypes []string, scannedRow []interface{}) []interface{} {
	row := make([]interface{}, len(scannedRow))
	for i, value := range scannedRow {
		// Unwrap the nullable scan destination into Grafana's nullable field value.
		switch v := value.(type) {
		case **bool:
			value = *v
		case **int64:
			value = *v
		case **float64:
			value = *v
		case **string:
			value = *v
		case **time.Time:
			value = *v
		}
		switch databaseTypes[i] {
		case "VARBINARY", "LONG VARBINARY", "BINARY":
			raw := *value.(*[]byte)
			if raw == nil {
				row[i] = (*string)(nil)
			} else {
				encoded := hex.EncodeToString(raw)
				row[i] = &encoded
			}
		default:
			if text, ok := value.(*string); ok && text != nil && !utf8.ValidString(*text) {
				validText := strings.ToValidUTF8(*text, "\uFFFD")
				row[i] = &validText
			} else {
				row[i] = value
			}
		}
	}
	return row
}
