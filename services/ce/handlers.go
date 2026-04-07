package ce

import (
	"net/http"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func jsonOK(body any) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK, Body: body, Format: service.FormatJSON}, nil
}

func jsonErr(awsErr *service.AWSError) (*service.Response, error) {
	return &service.Response{Format: service.FormatJSON}, awsErr
}

func str(params map[string]any, key string) string {
	if v, ok := params[key].(string); ok {
		return v
	}
	return ""
}

func handleGetCostAndUsage(params map[string]any, store *Store) (*service.Response, error) {
	timePeriod, _ := params["TimePeriod"].(map[string]any)
	start, _ := timePeriod["Start"].(string)
	end, _ := timePeriod["End"].(string)
	if start == "" || end == "" {
		return jsonErr(service.ErrValidation("TimePeriod.Start and TimePeriod.End are required"))
	}

	granularity := str(params, "Granularity")
	if granularity == "" {
		granularity = "DAILY"
	}

	var groupBy []map[string]string
	if gb, ok := params["GroupBy"].([]any); ok {
		for _, g := range gb {
			if gm, ok := g.(map[string]any); ok {
				entry := make(map[string]string)
				for k, v := range gm {
					if sv, ok := v.(string); ok {
						entry[k] = sv
					}
				}
				groupBy = append(groupBy, entry)
			}
		}
	}

	results := store.GenerateCostAndUsage(start, end, granularity, groupBy)

	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		entry := map[string]any{
			"TimePeriod": r.TimePeriod,
			"Total":      r.Total,
		}
		if len(r.Groups) > 0 {
			groups := make([]map[string]any, 0, len(r.Groups))
			for _, g := range r.Groups {
				groups = append(groups, map[string]any{
					"Keys":    g.Keys,
					"Metrics": g.Metrics,
				})
			}
			entry["Groups"] = groups
		}
		out = append(out, entry)
	}

	return jsonOK(map[string]any{
		"ResultsByTime":           out,
		"DimensionValueAttributes": []any{},
	})
}

func handleGetCostForecast(params map[string]any, store *Store) (*service.Response, error) {
	timePeriod, _ := params["TimePeriod"].(map[string]any)
	start, _ := timePeriod["Start"].(string)
	end, _ := timePeriod["End"].(string)
	if start == "" || end == "" {
		return jsonErr(service.ErrValidation("TimePeriod.Start and TimePeriod.End are required"))
	}

	granularity := str(params, "Granularity")
	if granularity == "" {
		granularity = "MONTHLY"
	}
	metric := str(params, "Metric")
	if metric == "" {
		metric = "UNBLENDED_COST"
	}

	results, total := store.GenerateForecast(start, end, granularity, metric)

	out := make([]map[string]any, 0, len(results))
	for _, r := range results {
		out = append(out, map[string]any{
			"TimePeriod": r.TimePeriod,
			"MeanValue":  r.Total[metric]["Amount"],
		})
	}

	return jsonOK(map[string]any{
		"Total": map[string]any{
			"Amount": total,
			"Unit":   "USD",
		},
		"ForecastResultsByTime": out,
	})
}

func handleGetDimensionValues(params map[string]any, store *Store) (*service.Response, error) {
	dimension := str(params, "Dimension")
	if dimension == "" {
		return jsonErr(service.ErrValidation("Dimension is required"))
	}

	values := store.GetDimensionValues(dimension)
	out := make([]map[string]any, 0, len(values))
	for _, v := range values {
		out = append(out, map[string]any{
			"Value":      v,
			"Attributes": map[string]any{},
		})
	}

	return jsonOK(map[string]any{
		"DimensionValues":  out,
		"ReturnSize":       len(values),
		"TotalSize":        len(values),
	})
}

func handleGetTags(store *Store) (*service.Response, error) {
	tags := store.GetTags()
	return jsonOK(map[string]any{
		"Tags":        tags,
		"ReturnSize":  len(tags),
		"TotalSize":   len(tags),
	})
}

func handleGetReservationUtilization(params map[string]any, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"UtilizationsByTime": []map[string]any{},
		"Total": map[string]any{
			"UtilizationPercentage":                "78.5",
			"PurchasedHours":                       "8760",
			"TotalActualHours":                     "6867.6",
			"UnusedHours":                          "1892.4",
			"OnDemandCostOfRIHoursUsed":            "2456.78",
			"NetRISavings":                         "1234.56",
			"TotalPotentialRISavings":              "1890.12",
			"AmortizedUpfrontFee":                  "500.00",
			"AmortizedRecurringFee":                "350.00",
			"TotalAmortizedFee":                    "850.00",
			"RICostForUnusedHours":                 "655.56",
			"RealizedSavings":                      "1234.56",
			"UnrealizedSavings":                    "655.56",
		},
	})
}

func handleGetSavingsPlansUtilization(params map[string]any, store *Store) (*service.Response, error) {
	return jsonOK(map[string]any{
		"SavingsPlansUtilizationsByTime": []map[string]any{},
		"Total": map[string]any{
			"Utilization": map[string]any{
				"TotalCommitment":     "500.00",
				"UsedCommitment":      "425.00",
				"UnusedCommitment":    "75.00",
				"UtilizationPercentage": "85.0",
			},
			"Savings": map[string]any{
				"NetSavings":          "187.50",
				"OnDemandCostEquivalent": "612.50",
			},
			"AmortizedCommitment": map[string]any{
				"AmortizedUpfrontCommitment": "200.00",
				"AmortizedRecurringCommitment": "300.00",
				"TotalAmortizedCommitment":    "500.00",
			},
		},
	})
}
