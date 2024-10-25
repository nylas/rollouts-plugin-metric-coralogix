package plugin

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/argoproj/argo-rollouts/utils/plugin/types"
	"github.com/opensearch-project/opensearch-go"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/argoproj/argo-rollouts/metricproviders/plugin"
	"github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/argoproj/argo-rollouts/utils/evaluate"
	metricutil "github.com/argoproj/argo-rollouts/utils/metric"
	timeutil "github.com/argoproj/argo-rollouts/utils/time"
	log "github.com/sirupsen/logrus"
)

// Here is a real implementation of MetricsPlugin
type RpcPlugin struct {
	LogCtx log.Entry
	api    v1.API
}

type Config struct {
	// Address is the HTTP address and port of the opensearch server
	Address string `json:"address,omitempty" protobuf:"bytes,1,opt,name=address"`
	// Username is the username to authenticate with
	Username string `json:"username,omitempty" protobuf:"bytes,2,opt,name=username"`
	// Username is password to authenticate with
	Password string `json:"password,omitempty" protobuf:"bytes,3,opt,name=password"`
	// InsecureSkipVerify skips the certificate verification step when set to true
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty" protobuf:"bytes,4,opt,name=insecureSkipVerify"`
	// Query is a raw opensearch query to perform
	Index string `json:"index,omitempty" protobuf:"bytes,5,opt,name=index"`
	// Query is a raw opensearch query to perform
	Query string `json:"query,omitempty" protobuf:"bytes,6,opt,name=query"`
}

func (g *RpcPlugin) InitPlugin() types.RpcError {
	return types.RpcError{}
}

func (g *RpcPlugin) Run(anaysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric) v1alpha1.Measurement {
	startTime := timeutil.MetaNow()
	newMeasurement := v1alpha1.Measurement{
		StartedAt: &startTime,
	}

	config := Config{}
	err := json.Unmarshal(metric.Provider.Plugin["argoproj-labs/opensearch-metric-plugin"], &config)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	api, err := newOpensearchAPI(config)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := newQuery(ctx, api, config.Index, config.Query)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	newValue, newStatus, err := g.processResponse(metric, response)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)

	}
	newMeasurement.Value = newValue
	newMeasurement.Phase = newStatus
	finishedTime := timeutil.MetaNow()
	newMeasurement.FinishedAt = &finishedTime
	return newMeasurement
}

func (g *RpcPlugin) Resume(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	return measurement
}

func (g *RpcPlugin) Terminate(analysisRun *v1alpha1.AnalysisRun, metric v1alpha1.Metric, measurement v1alpha1.Measurement) v1alpha1.Measurement {
	return measurement
}

func (g *RpcPlugin) GarbageCollect(*v1alpha1.AnalysisRun, v1alpha1.Metric, int) types.RpcError {
	return types.RpcError{}
}

func (g *RpcPlugin) Type() string {
	return plugin.ProviderType
}

func (g *RpcPlugin) GetMetadata(metric v1alpha1.Metric) map[string]string {
	metricsMetadata := make(map[string]string)

	config := Config{}
	json.Unmarshal(metric.Provider.Plugin["argoproj-labs/opensearch-metric-plugin"], &config)
	if config.Query != "" {
		metricsMetadata["ResolvedPrometheusQuery"] = config.Query
	}
	return metricsMetadata
}

func (g *RpcPlugin) processResponse(metric v1alpha1.Metric, response []interface{}) (string, v1alpha1.AnalysisPhase, error) {
	results := make([]float64, 0, len(response))
	valueStr := "["
	for _, b := range response {
		if b != nil {
			val, ok := b.(map[string]interface{})["doc_count"].(float64)
			if !ok {
				return "", v1alpha1.AnalysisPhaseError, errors.New("doc_count is not of type float64")
			}

			valueStr = valueStr + strconv.FormatFloat(val, 'f', -1, 64) + ","
			results = append(results, val)
		}
	}

	if len(valueStr) > 1 {
		valueStr = valueStr[:len(valueStr)-1]
	}
	valueStr = valueStr + "]"
	newStatus, err := evaluate.EvaluateResult(results, metric, g.LogCtx)
	return valueStr, newStatus, err
}

func newOpensearchAPI(config Config) (*opensearch.Client, error) {

	if len(config.Address) != 0 {
		if !isUrl(config.Address) {
			return nil, errors.New("opensearch address is not is url format")
		}
	} else {
		return nil, errors.New("opensearch address is not configured")
	}

	osConfig := opensearch.Config{
		Addresses: []string{config.Address},
	}

	if len(config.Username) != 0 {
		osConfig.Username = config.Username
	}

	if len(config.Password) != 0 {
		osConfig.Password = config.Password
	}

	if config.InsecureSkipVerify {
		osConfig.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}

	client, err := opensearch.NewClient(osConfig)

	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating the client: %s", err))
	}

	res, err := client.Info()
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error getting response: %s", err))
	}

	defer res.Body.Close()

	if res.IsError() {
		return nil, errors.New(fmt.Sprintf("error: %s", res.String()))
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, errors.New(fmt.Sprintf("error parsing the response body: %s", err))
	}

	return client, nil
}

func newQuery(ctx context.Context, client *opensearch.Client, index string, query string) ([]interface{}, error) {
	var (
		r map[string]interface{}
	)

	res, err := client.Search(
		client.Search.WithContext(ctx),
		client.Search.WithIndex(index),
		client.Search.WithBody(strings.NewReader(query)),
		client.Search.WithTrackTotalHits(true),
		client.Search.WithPretty(),
	)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error getting response: %s", err))
	}

	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return nil, errors.New(fmt.Sprintf("error parsing the response body: %s", err))
		} else {
			return nil, errors.New(fmt.Sprintf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			))
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, errors.New(fmt.Sprintf("error parsing the response body: %s", err))
	}

	log.Debugf(
		"[%s] %d hits; took: %dms",
		res.Status(),
		int(r["hits"].(map[string]interface{})["total"].(map[string]interface{})["value"].(float64)),
		int(r["took"].(float64)),
	)

	return r["aggregations"].(map[string]interface{})["logs_per_5min"].(map[string]interface{})["buckets"].([]interface{}), nil
}

func isUrl(str string) bool {
	u, err := url.Parse(str)
	if err != nil {
		log.Errorf("Error in parsing url: %v", err)
	}
	log.Debugf("Parsed url: %v", u)
	return err == nil && u.Scheme != "" && u.Host != ""
}
