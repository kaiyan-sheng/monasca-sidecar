// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateAvg(t *testing.T) {
	metricDimension := DimensionList{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimension}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimension}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// (2 + 1) / 2 = 1.5
	avgMetricString := calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, avgRule)
	assert.Equal(t, "# HELP avgRuleTestName\n# TYPE gauge \navgRuleTestName 1.500000e+00\n", avgMetricString)
}

func TestCalculateAvgNegative(t *testing.T) {
	metricDimension := DimensionList{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "-1.0", Dimensions: metricDimension}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "-2.0", Dimensions: metricDimension}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// (-1 - 2) / 2 = -1.5
	avgMetricString := calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, avgRule)
	assert.Equal(t, "# HELP avgRuleTestName\n# TYPE gauge \navgRuleTestName -1.500000e+00\n", avgMetricString)
}

func TestCalculateAvgWithDimensions(t *testing.T) {
	newMetricDimensions := []Dimension{}
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key2", Value: "value2"})
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key1", Value: "value1"})
	oldMetricDimensions := []Dimension{}
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key2", Value: "value2"})
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key1", Value: "value1"})

	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: newMetricDimensions, DimensionHash: convertDimensionsToHash(newMetricDimensions)}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: oldMetricDimensions, DimensionHash: convertDimensionsToHash(oldMetricDimensions)}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define queryInterval and avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// (2 + 1) / 2 = 1.5
	avgMetricString := calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, avgRule)
	assert.Equal(t, "# HELP avgRuleTestName\n# TYPE gauge \navgRuleTestName{key2=value2,key1=value1} 1.500000e+00\n", avgMetricString)
}

func TestCalculateAvgWithMisMatchDimensions(t *testing.T) {
	newMetricDimensions := []Dimension{}
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key1", Value: "value1"})
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key2", Value: "value2"})
	oldMetricDimensions := []Dimension{}
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key3", Value: "value3"})
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key4", Value: "value4"})

	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: newMetricDimensions, DimensionHash: convertDimensionsToHash(newMetricDimensions)}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: oldMetricDimensions, DimensionHash: convertDimensionsToHash(oldMetricDimensions)}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define queryInterval and avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// mismatch dimensions
	avgMetricString := calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, avgRule)
	assert.Equal(t, "", avgMetricString)
}

func TestCalculateAvgWithBadValueString(t *testing.T) {
	newMetricDimensions := []Dimension{}
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key1", Value: "value1"})
	newMetricDimensions = append(newMetricDimensions, Dimension{Key: "key2", Value: "value2"})
	oldMetricDimensions := []Dimension{}
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key3", Value: "value3"})
	oldMetricDimensions = append(oldMetricDimensions, Dimension{Key: "key4", Value: "value4"})

	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "abc", Dimensions: newMetricDimensions}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "def", Dimensions: oldMetricDimensions}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define queryInterval and avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// bad values
	avgMetricString := calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, avgRule)
	assert.Equal(t, "", avgMetricString)
}

func TestStructNewStringAvg(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_avg", Value: "2.0", Dimensions: newMetricDimension}
	avgValue := 1.0
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}
	stringAvg := structNewStringAvg(newPrometheusMetric, avgValue, avgRule)
	assert.Equal(t,
		"# HELP avgRuleTestName\n# TYPE gauge \navgRuleTestName 1.000000e+00\n",
		stringAvg)
}
