package daemon

import (
	"context"
	"fmt"
	"time"

	"github.com/scttfrdmn/prism/pkg/project"
	"github.com/scttfrdmn/prism/pkg/types"
)

// Phase 3e (#655): the budget forecast is computed here, in the daemon, over the real budgetengine
// spend ledger — replacing project.Manager.GetProjectForecast, which had no ledger and always ran the
// predictor on an empty history (so it projected all-zeros). Mirrors budgetStatus/costBreakdown: the
// daemon owns ledger analytics; pkg/project keeps the pure, history-driven BudgetPredictor.
//
// forecastHistoryDays bounds how much trailing ledger history feeds the regression. 180 days gives the
// linear-regression slope a stable base while staying comfortably larger than the default 6-month
// horizon; the predictor tolerates shorter spans (lower confidence).
const forecastHistoryDays = 180

// forecast builds a ledger-backed budget forecast for a project. It loads the project, folds the
// trailing ledger series, runs the existing BudgetPredictor, and maps the ShortfallPrediction into the
// wire-compatible ProjectForecastResponse (the exact mapping the Manager used, preserved verbatim).
func (s *Server) forecast(ctx context.Context, projectID string, req *project.ProjectForecastRequest) (*project.ProjectForecastResponse, error) {
	proj, err := s.projectManager.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	months := req.Months
	if months <= 0 {
		months = 6
	}

	// Real ledger-derived history (Phase 3e). Empty is fine — the predictor falls back to a rate-only
	// projection, exactly the prior behavior, just now non-empty whenever the ledger has spend.
	now := time.Now()
	history, err := s.ledgerCostSeries(ctx, projectID, now.AddDate(0, 0, -forecastHistoryDays), now)
	if err != nil {
		return nil, fmt.Errorf("build cost history: %w", err)
	}

	budget := proj.Budget
	if budget == nil {
		budget = &types.ProjectBudget{}
	}

	predictor := project.NewBudgetPredictor()
	prediction, err := predictor.Predict(projectID, history, budget, months, nil)
	if err != nil {
		return nil, fmt.Errorf("forecast failed: %w", err)
	}

	// Map ShortfallPrediction → legacy ProjectForecastResponse shape (unchanged wire contract).
	forecastData := make([]project.ForecastDataPoint, 0, len(prediction.MonthlyForecasts))
	for _, fm := range prediction.MonthlyForecasts {
		if fm.IsProjected {
			forecastData = append(forecastData, project.ForecastDataPoint{
				Month:          fm.Month,
				ProjectedCost:  fm.ProjectedSpend,
				CumulativeCost: fm.CumulativeSpend,
			})
		}
	}

	var historicalData []project.ForecastDataPoint
	if req.IncludeHistorical {
		for _, fm := range prediction.MonthlyForecasts {
			if !fm.IsProjected {
				cost := fm.ProjectedSpend
				if fm.ActualSpend != nil {
					cost = *fm.ActualSpend
				}
				historicalData = append(historicalData, project.ForecastDataPoint{
					Month:          fm.Month,
					ProjectedCost:  cost,
					CumulativeCost: fm.CumulativeSpend,
				})
			}
		}
	}

	confidence := 0.5
	switch prediction.ConfidenceLevel {
	case "high":
		confidence = 0.90
	case "medium":
		confidence = 0.75
	}

	return &project.ProjectForecastResponse{
		ProjectID:           projectID,
		GeneratedAt:         time.Now(),
		CurrentMonthlyRate:  prediction.CurrentDailyRate * 30,
		ForecastData:        forecastData,
		HistoricalData:      historicalData,
		ProjectedExhaustion: prediction.PredictedExhaustionAt,
		Confidence:          confidence,
	}, nil
}
