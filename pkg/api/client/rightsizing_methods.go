package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/scttfrdmn/prism/pkg/types"
)

// AnalyzeRightsizing performs rightsizing analysis for a specific instance
func (c *HTTPClient) AnalyzeRightsizing(ctx context.Context, req types.RightsizingAnalysisRequest) (*types.RightsizingAnalysisResponse, error) {
	resp, err := c.makeRequest(ctx, "POST", "/api/v1/rightsizing/analyze", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response types.RightsizingAnalysisResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetRightsizingRecommendations retrieves rightsizing recommendations for all instances
func (c *HTTPClient) GetRightsizingRecommendations(ctx context.Context) (*types.RightsizingRecommendationsResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/rightsizing/recommendations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response types.RightsizingRecommendationsResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetRightsizingStats retrieves detailed rightsizing statistics for an instance
func (c *HTTPClient) GetRightsizingStats(ctx context.Context, instanceName string) (*types.RightsizingStatsResponse, error) {
	if instanceName == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// Build URL with query parameters
	path := "/api/v1/rightsizing/stats?instance=" + url.QueryEscape(instanceName)

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response types.RightsizingStatsResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// ExportRightsizingData exports usage data for a specific instance
func (c *HTTPClient) ExportRightsizingData(ctx context.Context, instanceName string) ([]types.InstanceMetrics, error) {
	if instanceName == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// Build URL with query parameters
	path := "/api/v1/rightsizing/export?instance=" + url.QueryEscape(instanceName)

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metrics []types.InstanceMetrics
	if err := c.handleResponse(resp, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// GetRightsizingSummary retrieves fleet-wide rightsizing summary
func (c *HTTPClient) GetRightsizingSummary(ctx context.Context) (*types.RightsizingSummaryResponse, error) {
	resp, err := c.makeRequest(ctx, "GET", "/api/v1/rightsizing/summary", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var response types.RightsizingSummaryResponse
	if err := c.handleResponse(resp, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

// GetInstanceMetrics retrieves metrics for a specific instance
func (c *HTTPClient) GetInstanceMetrics(ctx context.Context, instanceName string, limit int) ([]types.InstanceMetrics, error) {
	if instanceName == "" {
		return nil, fmt.Errorf("instance name is required")
	}

	// Build URL path
	path := fmt.Sprintf("/api/v1/instances/%s/metrics", url.PathEscape(instanceName))

	// Add limit parameter if specified
	if limit > 0 {
		path += "?limit=" + strconv.Itoa(limit)
	}

	resp, err := c.makeRequest(ctx, "GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var metrics []types.InstanceMetrics
	if err := c.handleResponse(resp, &metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}
