package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/data"
	_ "github.com/vertica/vertica-sql-go"
)

// newDatasource returns datasource.ServeOpts.
func newDatasource() datasource.ServeOpts {
	// creates a instance manager for your plugin. The function passed
	// into `NewInstanceManger` is called when the instance is created
	// for the first time or when a datasource configuration changed.
	im := datasource.NewInstanceManager(newDataSourceInstance)
	ds := &VerticaDatasource{
		im: im,
	}

	return datasource.ServeOpts{
		QueryDataHandler:   ds,
		CheckHealthHandler: ds,
	}
}

// VerticaDatasource is an example datasource used to scaffold
// new datasource plugins with an backend.
type VerticaDatasource struct {
	// The instance manager can help with lifecycle management
	// of datasource instances in plugins. It's not a requirements
	// but a best practice that we recommend that you follow.
	im instancemgmt.InstanceManager
}

// GetVerticaDb will return the vertica db connection
// stored in the instance setting when the instance is created or update
func (td *VerticaDatasource) GetVerticaDb(pluginContext backend.PluginContext) (*sql.DB, error) {
	instance, err := td.im.Get(context.Background(), pluginContext)
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("getVerticaDb: %s", err))
		return nil, err
	}
	if instanceSetting, ok := instance.(*instanceSettings); ok {
		return instanceSetting.Db, nil
	} else {
		log.DefaultLogger.Info(fmt.Sprintf("getVerticaDb: %s", err))
		return nil, err
	}

}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifer).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (td *VerticaDatasource) QueryData(ctx context.Context, req *backend.QueryDataRequest) (*backend.QueryDataResponse, error) {
	// create response struct
	var (
		response = NewResponse(backend.NewQueryDataResponse())
		wg       = sync.WaitGroup{}
	)

	wg.Add(len(req.Queries))

	db, err := td.GetVerticaDb(req.PluginContext)
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :getVerticaDB: %s", err))
		return nil, err
	}

	// loop over queries and execute them individually.
	// added waitGroup to concurrently execute queries

	for _, q := range req.Queries {
		go func(query backend.DataQuery) {
			res := td.query(ctx, query, db)
			response.Set(query.RefID, res)
			wg.Done()
		}(q)

	}
	wg.Wait()

	return response.Response(), nil
}

type queryModel struct {
	QueryString     string  `json:"queryString"`
	QueryTemplated  string  `json:"queryTemplated,omitempty"`
	Hide            bool    `json:"hide,omitempty"`
	RefId           string  `json:"refId,omitempty"`
	TimeFillEnabled bool    `json:"timeFillEnabled,omitempty"`
	TimeFillMode    string  `json:"timeFillMode,omitempty"`
	TimeFillValue   float64 `json:"timeFillStaticValue,omitempty"`
	QueryType       string  `json:"format,omitempty"`
	IntervalMs      int     `json:"intervalMs,omitempty"`
	From            time.Time
	To              time.Time
}

func (td *VerticaDatasource) query(ctx context.Context, query backend.DataQuery, db *sql.DB) backend.DataResponse {
	// Unmarshal the json into queryModel type
	var qm queryModel

	response := backend.DataResponse{}
	err := json.Unmarshal(query.JSON, &qm)
	//Adding the from and To time ranges
	qm.To = query.TimeRange.To
	qm.From = query.TimeRange.From

	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :getVerticaDB: %s", err))
		response.Error = err
		return response
	}
	// return empty response when query.hide == true
	// do not execute query when the query panel is hidden in UI
	if qm.Hide {
		return response
	}

	connection, err := db.Conn(ctx)
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :connection: %s", err))
		response.Error = err
		return response
	}

	defer connection.Close()
	//run query
	rows, err := connection.QueryContext(ctx, qm.QueryTemplated)
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :queryContext: %s", err))
		response.Error = err
		return response
	}
	defer rows.Close()

	//get the column names, columns will be use to added a header names to the data frame
	columns, err := rows.Columns()
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :columns: %s", err))
		response.Error = err
		return response
	}

	//get the column types, column type will be used to convert column from sql type to data frame type. (implemented in types.go generateFrameType)
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("queryData :columnTypes: %s", err))
		response.Error = err
		return response
	}
	databaseTypes := make([]string, len(columnTypes))
	for i, columnType := range columnTypes {
		databaseTypes[i] = columnType.DatabaseTypeName()
	}

	//most of the SQL data source will return mostly a long frame, if group by is used.
	//so by default a long frame will be created.
	//generate the column types, using the info from columnTypes.
	// use the name (refId in query json) of the query as frame name
	longFrame := data.NewFrameOfFieldTypes(query.RefID, 0, generateFrameType(columnTypes)...)
	//setting the header names to the frame , the names are same as return by the driver.
	longFrame.SetFieldNames(columns...)

	//scaning fro rows.
	for rows.Next() {
		//generateRowIn returns an []interface{}, but based on the columns type variable to the retrived type is added.
		rowIn := generateRowIn(columnTypes)
		err = rows.Scan(rowIn...)
		if err != nil {
			log.DefaultLogger.Info(fmt.Sprintf("queryData :row.Scan: %s", err))
			response.Error = err
			return response
		}

		//append the scanned rows to the frame.
		longFrame.AppendRow(prepareRowForFrame(databaseTypes, rowIn)...)

	}
	//will use the queryType parameter from query to format the time series
	switch qm.QueryType {
	case "Time Series":
		// is 0 rows received , return an empty long frame
		if longFrame.Rows() == 0 {
			response.Frames = append(response.Frames, data.NewFrame(query.RefID))
		} else {
			//check of for frame type if not wide convert it to wide , when the query Type is time series.
			if longFrame.TimeSeriesSchema().Type != data.TimeSeriesTypeWide {
				longFrame, err = data.LongToWide(longFrame, nil)
				if err != nil {
					log.DefaultLogger.Info(fmt.Sprintf("queryData :LongToWide: %s", err))
					response.Error = err
					return response
				}
			}

			//fill time gaps if TimeFillEnabled is true
			if qm.TimeFillEnabled {
				frame, err := TimeGapFill(longFrame, qm)
				if err != nil {
					log.DefaultLogger.Info(fmt.Sprintf("queryData :TimeGapFill: %s", err))
					response.Error = err
					return response
				}
				fillMissing := &data.FillMissing{}
				switch qm.TimeFillMode {
				case "static":
					fillMissing.Mode = data.FillModeValue
					fillMissing.Value = qm.TimeFillValue
				case "null":
					fillMissing.Mode = data.FillModeNull
				case "previous":
					fillMissing.Mode = data.FillModePrevious
				default:
					fillMissing.Mode = data.FillModeNull
				}

				for _, field := range frame.Fields {
					if field.Type() != data.FieldTypeTime && field.Type() != data.FieldTypeNullableTime {
						for rowIdx := 0; rowIdx < frame.Rows(); rowIdx++ {
							var previousRow int
							if rowIdx == 0 {
								previousRow = 0
							} else {
								previousRow = rowIdx - 1
							}
							if _, ok := field.ConcreteAt(rowIdx); !ok {
								filled, err := data.GetMissing(fillMissing, field, previousRow)
								if err != nil {
									log.DefaultLogger.Info(fmt.Sprintf("queryData :fillmissing : %s", err))
								} else {
									field.Set(rowIdx, filled)
								}
							}

						}

					}

				}
				response.Frames = append(response.Frames, frame)
			} else {
				response.Frames = append(response.Frames, longFrame)
			}

		}
	default:
		//response for rest of the query types a long frame
		if longFrame.Rows() == 0 {
			response.Frames = append(response.Frames, data.NewFrame(query.RefID))
		} else {
			response.Frames = append(response.Frames, longFrame)
		}

	}
	return response
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (td *VerticaDatasource) CheckHealth(ctx context.Context, req *backend.CheckHealthRequest) (*backend.CheckHealthResult, error) {
	var status = backend.HealthStatusOk

	db, err := td.GetVerticaDb(req.PluginContext)
	if err != nil {
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("%s", err),
		}, nil
	}
	// https://golang.org/pkg/database/sql/#DBStats
	log.DefaultLogger.Info(fmt.Sprintf("stats before query: %v", db.Stats()))
	connection, err := db.Conn(ctx)
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("CheckHealth :connection: %s", err))
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("%s", err),
		}, nil
	}
	defer connection.Close()
	result, err := connection.QueryContext(ctx, "SELECT version()")
	if err != nil {
		log.DefaultLogger.Info(fmt.Sprintf("CheckHealth :QueryContext: %s", err))
		return &backend.CheckHealthResult{
			Status:  backend.HealthStatusError,
			Message: fmt.Sprintf("%s", err),
		}, nil
	}
	defer result.Close()
	var queryResult string

	if result.Next() {
		err = result.Scan(&queryResult)
		if err != nil {
			log.DefaultLogger.Info(fmt.Sprintf("CheckHealth :Scan: %s", err))
			return &backend.CheckHealthResult{
				Status:  backend.HealthStatusError,
				Message: fmt.Sprintf("%s", err),
			}, nil
		}
	}
	// https://golang.org/pkg/database/sql/#DBStats
	log.DefaultLogger.Info(fmt.Sprintf("CheckHealth: stats after query: %v", db.Stats()))
	return &backend.CheckHealthResult{
		Status:  status,
		Message: fmt.Sprintf("Successfully connected to %s", queryResult),
	}, nil
}

type instanceSettings struct {
	httpClient *http.Client
	Db         *sql.DB
	Name       string
}

func newDataSourceInstance(_ context.Context, settings backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	// func newDataSourceInstance(setting backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	var config datasourceConfig
	secret := settings.DecryptedSecureJSONData["password"]

	err := json.Unmarshal(settings.JSONData, &config)
	if err != nil {
		return nil, err
	}
	connStr := config.ConnectionURL(secret)
	db, err := sql.Open("vertica", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(config.MaxOpenConnections)
	db.SetMaxIdleConns(config.MaxIdealConnections)
	db.SetConnMaxIdleTime(time.Minute * time.Duration(config.MaxConnectionIdealTime))
	log.DefaultLogger.Info(fmt.Sprintf("newDataSourceInstance: new instance of datasource created: %+v", settings.Name))
	return &instanceSettings{
		httpClient: &http.Client{},
		Db:         db,
		Name:       settings.Name,
	}, nil

}

func (s *instanceSettings) Dispose() {
	s.Db.Close()
	log.DefaultLogger.Info(fmt.Sprintf("db connections of datasource %s closed", s.Name))
}
