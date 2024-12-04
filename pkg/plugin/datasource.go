package plugin

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"

	"github.com/academo/wasmtest/pkg/models"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/data"
)

//go:embed module.js
var moduleJs string

// Make sure Datasource implements required interfaces. This is important to do
// since otherwise we will only get a not implemented error response from plugin in
// runtime. In this example datasource instance implements backend.QueryDataHandler,
// backend.CheckHealthHandler interfaces. Plugin should not implement all these
// interfaces - only those which are required for a particular task.
var (
	_ backend.QueryDataHandler      = (*Datasource)(nil)
	_ backend.CheckHealthHandler    = (*Datasource)(nil)
	_ instancemgmt.InstanceDisposer = (*Datasource)(nil)
)

// NewDatasource creates a new datasource instance.
func NewDatasource(
	_ context.Context,
	_ backend.DataSourceInstanceSettings,
) (instancemgmt.Instance, error) {
	vm := goja.New()
	new(require.Registry).Enable(vm)
	console.Enable(vm)

	return &Datasource{
		goja: vm,
	}, nil
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	goja *goja.Runtime
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
}

// todo return string, err instead of panics
func (d *Datasource) queryFeDS(q backend.QueryDataRequest) string {
	//sleep 3 seconds
	backend.Logger.Info("testMe")

	execute := strings.ReplaceAll(moduleJs, "//# sourceMappingURL=module.js.map", "")
	// replace define calls with gojaDefine
	execute = strings.ReplaceAll(execute, "define([\"", "gojaDefine([\"")

	_, err := d.goja.RunString(execute)
	if err != nil {
		panic(err)
	}

	runQuery, ok := goja.AssertFunction(d.goja.Get("runQuery"))
	if !ok {
		panic("Not a function")
	}

	var qm queryModel
	_ = json.Unmarshal(q.Queries[0].JSON, &qm)
	query := FrontendQueryModel{
		App:       "dashboard",
		RequestID: "SQR101",
		Timezone:  "browser",
		Range: TimeRange{
			From: q.Queries[0].TimeRange.From.Format(time.RFC3339),
			To:   q.Queries[0].TimeRange.To.Format(time.RFC3339),
			Raw: struct {
				From string `json:"from"`
				To   string `json:"to"`
			}{
				From: q.Queries[0].TimeRange.From.Format(time.RFC3339),
				To:   q.Queries[0].TimeRange.To.Format(time.RFC3339),
			},
		},
		Interval:   "30s",
		IntervalMs: 30000,
		Targets: []Target{
			{
				Constant: parsedConstant(qm.Constant),
				DataSource: DataSource{
					Type: q.PluginContext.DataSourceInstanceSettings.Type,
					UID:  q.PluginContext.DataSourceInstanceSettings.UID,
				},
				QueryText: qm.QueryText,
				RefID:     q.Queries[0].RefID,
			},
		},
	}

	queryJson, err := json.Marshal(query)
	if err != nil {
		panic(err)
	}

	value, err := runQuery(
		goja.Undefined(),
		d.goja.ToValue(string(queryJson)),
	)
	if err != nil {
		panic(err)
	}

	var result any
	if p, ok := value.Export().(*goja.Promise); ok {
		switch p.State() {
		//todo, handle rejected
		case goja.PromiseStateRejected:
			panic(p.Result().String())
		case goja.PromiseStateFulfilled:
			result = p.Result().Export()
		default:
			//todo handle unexpected
			panic("unexpected promise state pending")
		}
	}

	return result.(string)
}

// QueryData handles multiple queries and returns multiple responses.
// req contains the queries []DataQuery (where each query contains RefID as a unique identifier).
// The QueryDataResponse contains a map of RefID to the response for each query, and each response
// contains Frames ([]*Frame).
func (d *Datasource) QueryData(
	ctx context.Context,
	req *backend.QueryDataRequest,
) (*backend.QueryDataResponse, error) {

	backend.Logger.Info("QueryData", "req", req)
	rawFEresponse := d.queryFeDS(*req)
	backend.Logger.Info("I am returning the parsed FE response")
	parsed := convertJSResultToQueryDataResponse(rawFEresponse)
	backend.Logger.Info("parsed", parsed)
	return parsed, nil
}

func convertJSResultToQueryDataResponse(resultStr string) *backend.QueryDataResponse {
	response := backend.NewQueryDataResponse()

	var parsed struct {
		Data []struct {
			RefID  string `json:"refId"`
			Fields []struct {
				Name   string `json:"name"`
				Values []any  `json:"values"`
				Type   string `json:"type"`
			} `json:"fields"`
			Length int `json:"length"`
		} `json:"data"`
	}

	if err := json.Unmarshal([]byte(resultStr), &parsed); err != nil {
		// Since we need to return a QueryDataResponse, we'll add the error to the response
		response.Responses["A"] = backend.DataResponse{
			Error:  fmt.Errorf("json unmarshal: %v", err),
			Status: backend.StatusBadRequest,
		}
		return response
	}

	for _, d := range parsed.Data {
		frame := data.NewFrame(d.RefID)

		for _, f := range d.Fields {
			var field *data.Field
			switch f.Type {
			case "time":
				times := make([]time.Time, len(f.Values))
				for i, v := range f.Values {
					t, _ := time.Parse(time.RFC3339, v.(string))
					times[i] = t
				}
				field = data.NewField(f.Name, nil, times)
			case "number":
				numbers := make([]float64, len(f.Values))
				for i, v := range f.Values {
					numbers[i] = v.(float64)
				}
				field = data.NewField(f.Name, nil, numbers)
			case "string":
				strings := make([]string, len(f.Values))
				for i, v := range f.Values {
					strings[i] = v.(string)
				}
				field = data.NewField(f.Name, nil, strings)
			}
			frame.Fields = append(frame.Fields, field)
		}

		response.Responses[d.RefID] = backend.DataResponse{
			Frames: []*data.Frame{frame},
		}
	}

	return response
}

type queryModel struct {
	QueryText string `json:"queryText"`
	Constant  string `json:"constant"`
}

type TimeRange struct {
	From string `json:"from"`
	To   string `json:"to"`
	Raw  struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"raw"`
}

type DataSource struct {
	Type string `json:"type"`
	UID  string `json:"uid"`
}

type Target struct {
	Constant   float64    `json:"constant"`
	DataSource DataSource `json:"datasource"`
	QueryText  string     `json:"queryText"`
	RefID      string     `json:"refId"`
}

type FrontendQueryModel struct {
	App        string    `json:"app"`
	RequestID  string    `json:"requestId"`
	Timezone   string    `json:"timezone"`
	Range      TimeRange `json:"range"`
	Interval   string    `json:"interval"`
	IntervalMs int       `json:"intervalMs"`
	Targets    []Target  `json:"targets"`
}

// CheckHealth handles health checks sent from Grafana to the plugin.
// The main use case for these health checks is the test button on the
// datasource configuration page which allows users to verify that
// a datasource is working as expected.
func (d *Datasource) CheckHealth(
	_ context.Context,
	req *backend.CheckHealthRequest,
) (*backend.CheckHealthResult, error) {
	res := &backend.CheckHealthResult{}
	config, err := models.LoadPluginSettings(*req.PluginContext.DataSourceInstanceSettings)

	if err != nil {
		res.Status = backend.HealthStatusError
		res.Message = "Unable to load settings"
		return res, nil
	}

	if config.Secrets.ApiKey == "" {
		res.Status = backend.HealthStatusError
		res.Message = "API key is missing"
		return res, nil
	}

	return &backend.CheckHealthResult{
		Status:  backend.HealthStatusOk,
		Message: "Data source is working",
	}, nil
}

func parsedConstant(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
