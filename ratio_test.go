// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateRatio(t *testing.T) {
	metricDimensions := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions}
	prometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.2", Dimensions: metricDimensions}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)

	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	// 0.2 / 2.0 = 0.1
	avgMetricString := calculateRatio(prometheusMetrics, ratioRule)
	assert.Equal(t, "# HELP ratioRuleTestName\n# TYPE gauge \nratioRuleTestName 1.000000e-01\n", avgMetricString)
}

func TestCalculateRatioWithDimensions(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensions = append(metricDimensions, Dimension{Key: "key2", Value: "value2"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key4", Value: "value4"})

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
	prometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.2", Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
	prometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensionsDiff, DimensionHash: convertDimensionsToHash(metricDimensionsDiff)}
	prometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.1", Dimensions: metricDimensionsDiff, DimensionHash: convertDimensionsToHash(metricDimensionsDiff)}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric3)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric4)

	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	// (2 + 1) / 2 = 1.5
	avgMetricString := calculateRatio(prometheusMetrics, ratioRule)
	assert.Equal(t, "# HELP ratioRuleTestName\n# TYPE gauge \nratioRuleTestName{key1=value1,key2=value2} 1.000000e-01\n# HELP ratioRuleTestName\n# TYPE gauge \nratioRuleTestName{key3=value3,key4=value4} 2.000000e-02\n", avgMetricString)
}

func TestCalculateRatioWithMisMatchDimensions(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensions = append(metricDimensions, Dimension{Key: "key2", Value: "value2"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key4", Value: "value4"})

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
	prometheusMetric2 := PrometheusMetric{Name: "request_total_time", Value: "0.2", Dimensions: metricDimensionsDiff, DimensionHash: convertDimensionsToHash(metricDimensionsDiff)}
	prometheusMetric3 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
	prometheusMetric4 := PrometheusMetric{Name: "request_total_time", Value: "0.1", Dimensions: metricDimensionsDiff, DimensionHash: convertDimensionsToHash(metricDimensionsDiff)}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric3)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric4)

	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	// (2 + 1) / 2 = 1.5
	avgMetricString := calculateRatio(prometheusMetrics, ratioRule)
	assert.Equal(t, "", avgMetricString)
}

func TestFindDenominatorValue(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	numeratorDimHash := convertDimensionsToHash(metricDimensions)
	denominatorDimHash := convertDimensionsToHash(metricDimensionsDiff)

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: numeratorDimHash}
	prometheusMetric2 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensionsDiff, DimensionHash: denominatorDimHash}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)

	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	denominatorValue, errDenominator := findDenominatorValue(prometheusMetrics, numeratorDimHash, ratioRule)
	assert.Equal(t, 2.0, denominatorValue)
	assert.Equal(t, nil, errDenominator)
}

func TestFindDenominatorValueFailed(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	numeratorDimHash := convertDimensionsToHash(metricDimensions)
	denominatorDimHash := convertDimensionsToHash(metricDimensionsDiff)

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: denominatorDimHash}
	prometheusMetric2 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensionsDiff, DimensionHash: denominatorDimHash}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)

	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	denominatorValue, errDenominator := findDenominatorValue(prometheusMetrics, numeratorDimHash, ratioRule)
	assert.Equal(t, 0.0, denominatorValue)
	assert.NotEqual(t, nil, errDenominator)
}
