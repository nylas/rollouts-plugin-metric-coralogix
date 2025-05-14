package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/argoproj/argo-rollouts/utils/plugin/types"
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
	// BaseUrl is the base url of your Coralogix environment
	BaseUrl string `json:"baseUrl,omitempty" protobuf:"bytes,1,opt,name=baseUrl"`
	// APIKey is the API key to authenticate with
	APIKey string `json:"apiKey,omitempty" protobuf:"bytes,2,opt,name=apiKey"`
	// Query is the DataPrime query to execute
	Query string `json:"query,omitempty" protobuf:"bytes,3,opt,name=query"`
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
	err := json.Unmarshal(metric.Provider.Plugin["argoproj-labs/coralogix-metric-plugin"], &config)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	client, err := newCoralogixClient(config)
	if err != nil {
		return metricutil.MarkMeasurementError(newMeasurement, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := client.executeQuery(ctx, config.Query)
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
	json.Unmarshal(metric.Provider.Plugin["argoproj-labs/coralogix-metric-plugin"], &config)
	if config.Query != "" {
		metricsMetadata["ResolvedCoralogixQuery"] = config.Query
	}
	return metricsMetadata
}

func (g *RpcPlugin) processResponse(metric v1alpha1.Metric, response []interface{}) (string, v1alpha1.AnalysisPhase, error) {
	results := make([]float64, 0, len(response))
	valueStr := "["
	for _, b := range response {
		if b != nil {
			// Extract the ratio value from the userData
			userData, ok := b.(map[string]interface{})
			if !ok {
				return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("invalid userData format")
			}

			ratio, ok := userData["ratio"].(float64)
			if !ok {
				return "", v1alpha1.AnalysisPhaseError, fmt.Errorf("ratio is not of type float64")
			}

			valueStr = valueStr + strconv.FormatFloat(ratio, 'f', -1, 64) + ","
			results = append(results, ratio)
		}
	}

	if len(valueStr) > 1 {
		valueStr = valueStr[:len(valueStr)-1]
	}
	valueStr = valueStr + "]"
	newStatus, err := evaluate.EvaluateResult(results, metric, g.LogCtx)
	return valueStr, newStatus, err
}
