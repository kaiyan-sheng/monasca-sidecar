// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateDeltaRatio(t *testing.T) {
	metricDimensions := []Dimension{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}

	// define new prometheusMetrics
	newPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensions}
	newPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.3", Dimensions: metricDimensions}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric1)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric2)
	// define old prometheusMetrics
	oldPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimensions}
	oldPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.1", Dimensions: metricDimensions}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric1)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric2)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.3 - 0.1) / (5.0 - 1.0) = 0.05
	deltaRatioMetricString := calculateDeltaRatio(newPrometheusMetrics, oldPrometheusMetrics, deltaRatioRule)
	assert.Equal(t, "# HELP deltaRatioRuleTestName\n# TYPE gauge\ndeltaRatioRuleTestName 5.000000e-02\n", deltaRatioMetricString)
}

func TestCalculateDeltaDeltaRatioWithDimensions(t *testing.T) {
	metricDimensions1 := []Dimension{}
	metricDimensions2 := []Dimension{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key1", Value: "value1"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key2", Value: "value2"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key3", Value: "value3"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key4", Value: "value4"})
	dimensionHash1 := convertDimensionsToHash(metricDimensions1)
	dimensionHash2 := convertDimensionsToHash(metricDimensions2)

	// define new prometheusMetrics
	newPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	newPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.3", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	newPrometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "4.0", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	newPrometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.4", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric1)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric2)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric3)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric4)

	// define old prometheusMetrics
	oldPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	oldPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.1", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	oldPrometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	oldPrometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.2", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric1)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric2)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric3)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric4)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.3 - 0.1) / (5.0 - 1.0) = 0.05 - dim1
	// (0.4 - 0.2) / (4.0 - 2.0) = 0.1 - dim2
	deltaRatioMetricString := calculateDeltaRatio(newPrometheusMetrics, oldPrometheusMetrics, deltaRatioRule)
	expectedStringValue := `# HELP deltaRatioRuleTestName
# TYPE gauge
deltaRatioRuleTestName{key1=value1,key2=value2} 5.000000e-02
# HELP deltaRatioRuleTestName
# TYPE gauge
deltaRatioRuleTestName{key3=value3,key4=value4} 1.000000e-01
`
	assert.Equal(t, expectedStringValue, deltaRatioMetricString)
}

func TestCalculateDeltaDeltaRatioWithMisMatchDimensions(t *testing.T) {
	metricDimensions1 := []Dimension{}
	metricDimensions2 := []Dimension{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key1", Value: "value1"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key2", Value: "value2"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key3", Value: "value3"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key4", Value: "value4"})
	dimensionHash1 := convertDimensionsToHash(metricDimensions1)
	dimensionHash2 := convertDimensionsToHash(metricDimensions2)

	// define new prometheusMetrics
	newPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	newPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.3", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	newPrometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "4.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	newPrometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.4", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric1)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric2)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric3)
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric4)

	// define old prometheusMetrics
	oldPrometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	oldPrometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.1", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	oldPrometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions1, DimensionHash: dimensionHash1}
	oldPrometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.2", Dimensions: metricDimensions2, DimensionHash: dimensionHash2}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric1)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric2)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric3)
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric4)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	deltaRatioMetricString := calculateDeltaRatio(newPrometheusMetrics, oldPrometheusMetrics, deltaRatioRule)
	assert.Equal(t, "", deltaRatioMetricString)
}
