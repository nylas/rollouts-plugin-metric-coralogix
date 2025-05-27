package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CoralogixClient struct {
	baseUrl   string
	apiKey    string
	queryTier QueryTier
}

func newCoralogixClient(config Config) (*CoralogixClient, error) {
	// Default to TierFrequentSearch if not specified
	queryTier := config.QueryTier
	if queryTier == "" {
		queryTier = TierFrequentSearch
	}

	return &CoralogixClient{
		baseUrl:   config.BaseUrl,
		apiKey:    config.APIKey,
		queryTier: queryTier,
	}, nil
}

type QueryResponse struct {
	Result struct {
		Results []struct {
			Metadata []interface{} `json:"metadata"`
			Labels   []interface{} `json:"labels"`
			UserData string        `json:"userData"`
		} `json:"results"`
	} `json:"result"`
}

func (c *CoralogixClient) executeQuery(ctx context.Context, query string) ([]interface{}, error) {
	type queryRequest struct {
		Metadata struct {
			Tier QueryTier `json:"tier,omitempty"`
		} `json:"metadata"`
		Query string `json:"query"`
	}

	reqBody := queryRequest{
		Query: query,
	}
	reqBody.Metadata.Tier = c.queryTier

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseUrl+"/api/v1/dataprime/query", bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read the entire response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Split the NDJSON response into individual JSON objects
	jsonObjects := strings.Split(string(body), "\n")

	// Find the result object (second object in NDJSON)
	if len(jsonObjects) < 2 {
		return nil, fmt.Errorf("invalid response format: expected at least 2 JSON objects")
	}

	// Parse the result object
	var queryResp QueryResponse
	if err := json.Unmarshal([]byte(jsonObjects[1]), &queryResp); err != nil {
		return nil, fmt.Errorf("failed to parse result object: %w", err)
	}

	// Parse userData for each result
	results := make([]interface{}, len(queryResp.Result.Results))
	for i, result := range queryResp.Result.Results {
		var userData map[string]interface{}
		if err := json.Unmarshal([]byte(result.UserData), &userData); err != nil {
			return nil, fmt.Errorf("failed to parse userData: %w", err)
		}
		results[i] = userData
	}

	// Print the final processed results
	fmt.Printf("Coralogix query results: %+v\n", results)

	return results, nil
}
