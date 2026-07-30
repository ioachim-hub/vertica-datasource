package main

import (
	"encoding/json"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

var _ backend.QueryDataHandler = (*VerticaDatasource)(nil)
var _ backend.CheckHealthHandler = (*VerticaDatasource)(nil)

func TestNewDatasourceRegistersRequiredHandlers(t *testing.T) {
	serveOpts := newDatasource()

	queryHandler, ok := serveOpts.QueryDataHandler.(*VerticaDatasource)
	if !ok || queryHandler == nil {
		t.Fatal("newDatasource did not register a VerticaDatasource QueryData handler")
	}

	healthHandler, ok := serveOpts.CheckHealthHandler.(*VerticaDatasource)
	if !ok || healthHandler == nil {
		t.Fatal("newDatasource did not register a VerticaDatasource CheckHealth handler")
	}

	if queryHandler != healthHandler {
		t.Fatal("newDatasource registered different QueryData and CheckHealth handlers")
	}
}

func TestDatasourceConfigUnmarshalPreservesConnectionSettings(t *testing.T) {
	var config datasourceConfig
	err := json.Unmarshal([]byte(`{
		"database":"analytics",
		"host":"vertica.example.test:5433",
		"useConnectionLoadbalancing":true,
		"usePreparedStatement":true,
		"user":"grafana",
		"sslMode":"require",
		"maxOpenConnections":12,
		"maxIdealConnections":7,
		"maxConnectionIdealTime":45
	}`), &config)
	if err != nil {
		t.Fatalf("unmarshal datasource config: %v", err)
	}

	if config.Database != "analytics" || config.Host != "vertica.example.test:5433" ||
		!config.UseConnectionLoadbalancing || !config.UsePreparedStatement ||
		config.User != "grafana" || config.SSlMode != "require" ||
		config.MaxOpenConnections != 12 || config.MaxIdealConnections != 7 ||
		config.MaxConnectionIdealTime != 45 {
		t.Fatalf("datasource config did not preserve settings: %#v", config)
	}
}

func TestQueryModelUnmarshalPreservesQuerySettings(t *testing.T) {
	var query queryModel
	err := json.Unmarshal([]byte(`{
		"queryString":"SELECT * FROM metrics",
		"queryTemplated":"SELECT * FROM metrics WHERE ts >= $__from",
		"hide":true,
		"refId":"A",
		"timeFillEnabled":true,
		"timeFillMode":"value",
		"timeFillStaticValue":3.5,
		"format":"time_series",
		"intervalMs":60000
	}`), &query)
	if err != nil {
		t.Fatalf("unmarshal query model: %v", err)
	}

	if query.QueryString != "SELECT * FROM metrics" ||
		query.QueryTemplated != "SELECT * FROM metrics WHERE ts >= $__from" ||
		!query.Hide || query.RefId != "A" || !query.TimeFillEnabled ||
		query.TimeFillMode != "value" || query.TimeFillValue != 3.5 ||
		query.QueryType != "time_series" || query.IntervalMs != 60000 {
		t.Fatalf("query model did not preserve settings: %#v", query)
	}
}
