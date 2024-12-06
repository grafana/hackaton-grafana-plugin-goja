package plugin

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

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
	nodeCmd, err := startNodeProcess()
	if err != nil {
		return nil, err
	}

	//wait 2 seconds
	time.Sleep(2 * time.Second)

	return &Datasource{
		nodeCmd: nodeCmd,
	}, nil
}

func startNodeProcess() (*exec.Cmd, error) {
	execute := strings.ReplaceAll(moduleJs, "//# sourceMappingURL=module.js.map", "")
	// write somewhere to a temp file

	tmpFile, err := os.CreateTemp("", "module.js")
	if err != nil {
		return nil, err
	}

	//write the file
	if _, err := tmpFile.WriteString(execute); err != nil {
		return nil, err
	}
	tmpFile.Close()

	cmd := exec.Command("node", tmpFile.Name())
	cmd.Stdout = os.Stdout
	err = cmd.Start()

	return cmd, err
}

// Datasource is an example datasource which can respond to data queries, reports
// its health and has streaming skills.
type Datasource struct {
	nodeCmd *exec.Cmd
	socket  *websocket.Conn
}

// Dispose here tells plugin SDK that plugin wants to clean up resources when a new instance
// created. As soon as datasource settings change detected by SDK old datasource instance will
// be disposed and a new one will be created using NewSampleDatasource factory function.
func (d *Datasource) Dispose() {
	// Clean up datasource instance resources.
	err := d.nodeCmd.Process.Kill()
	if err != nil {
		panic(err)
	}
}

// todo return string, err instead of panics
func (d *Datasource) queryFeDS(q backend.QueryDataRequest) string {
	//sleep 3 seconds
	backend.Logger.Info("testMe")

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

	resp, err := http.Post("http://localhost:8080/query",
		"application/json",
		strings.NewReader(string(queryJson)))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(respBody)
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
	backend.Logger.Info("rawFEresponse", rawFEresponse)
	backend.Logger.Info("I am returning the parsed FE response")
	parsed := convertJSResultToQueryDataResponse(rawFEresponse)
	backend.Logger.Info("parsed", parsed)
	return parsed, nil
}

type todoField struct {
	Name   string `json:"name"`
	Values []any  `json:"values"`
	Type   string `json:"type"`
}

type todoData struct {
	RefID  string      `json:"refId"`
	Fields []todoField `json:"fields"`
}

type parsedResponse struct {
	Data []todoData `json:"data"`
}

func convertJSResultToQueryDataResponse(resultStr string) *backend.QueryDataResponse {
	response := backend.NewQueryDataResponse()
	var parsed parsedResponse

	if err := json.Unmarshal([]byte(resultStr), &parsed); err != nil {
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
			case "number":
				nums := make([]float64, len(f.Values))
				for i, v := range f.Values {
					nums[i] = v.(float64)
				}
				field = data.NewField(f.Name, nil, nums)
			case "string":
				strs := make([]string, len(f.Values))
				for i, v := range f.Values {
					strs[i] = v.(string)
				}
				field = data.NewField(f.Name, nil, strs)
			case "boolean":
				bools := make([]bool, len(f.Values))
				for i, v := range f.Values {
					bools[i] = v.(bool)
				}
				field = data.NewField(f.Name, nil, bools)
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
