package utils

import "backend/services"

// Optional helper function for parsing
func ParseChartConfigToChartJS(chartConfig *services.ChartConfiguration) (map[string]interface{}, map[string]interface{}) {
	chartJSConfig := map[string]interface{}{
		"labels": chartConfig.Labels,
		"datasets": []map[string]interface{}{
			{
				"label":           chartConfig.YLabel,
				"data":            chartConfig.Values,
				"backgroundColor": "rgba(59, 130, 246, 0.5)",
			},
		},
	}

	options := map[string]interface{}{
		"responsive": true,
		"plugins": map[string]interface{}{
			"title": map[string]interface{}{
				"display": true,
				"text":    chartConfig.Title,
			},
		},
		"scales": map[string]interface{}{
			"x": map[string]interface{}{
				"title": map[string]interface{}{
					"display": true,
					"text":    chartConfig.XLabel,
				},
			},
			"y": map[string]interface{}{
				"title": map[string]interface{}{
					"display": true,
					"text":    chartConfig.YLabel,
				},
			},
		},
	}

	return chartJSConfig, options
}
