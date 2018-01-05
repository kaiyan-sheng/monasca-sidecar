// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateRate(t *testing.T) {
	metricDimension := DimensionList{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimension}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimension}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (2 - 1) / 10.0 = 0.1
	rateMetricString := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rateRule)
	assert.Equal(t, "# HELP rateRuleTestName\n# TYPE gauge \nrateRuleTestName 1.000000e-01\n", rateMetricString)
}

func TestCalculateRateNegative(t *testing.T) {
	metricDimension := DimensionList{}
	newPrometheusMetrics := []PrometheusMetric{}
	oldPrometheusMetrics := []PrometheusMetric{}
	// define newPrometheusMetrics
	newPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "1.0", Dimensions: metricDimension}
	newPrometheusMetrics = append(newPrometheusMetrics, newPrometheusMetric)
	// define oldPrometheusMetrics
	oldPrometheusMetric := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimension}
	oldPrometheusMetrics = append(oldPrometheusMetrics, oldPrometheusMetric)
	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (1 - 2) / 10.0 = -0.1
	rateMetricString := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rateRule)
	assert.Equal(t, "# HELP rateRuleTestName\n# TYPE gauge \nrateRuleTestName -1.000000e-01\n", rateMetricString)
}

func TestCalculateRateWithDimensions(t *testing.T) {
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
	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (2 - 1) / 10.0 = 0.1
	rateMetricString := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rateRule)
	assert.Equal(t, "# HELP rateRuleTestName\n# TYPE gauge \nrateRuleTestName{key2=value2,key1=value1} 1.000000e-01\n", rateMetricString)
}

func TestCalculateRateWithMisMatchDimensions(t *testing.T) {
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
	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (2 - 1) / 10.0 = 0.1
	rateMetricString := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rateRule)
	assert.Equal(t, "", rateMetricString)
}

func TestCalculateRateWithBadValueString(t *testing.T) {
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
	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (2 - 1) / 10.0 = 0.1
	rateMetricString := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rateRule)
	assert.Equal(t, "", rateMetricString)
}

func TestStructNewStringRate(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "2.0", Dimensions: newMetricDimension}
	rateValue := 1.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}
	stringRate := structNewStringRate(newPrometheusMetric, rateValue, rateRule)
	assert.Equal(t,
		"# HELP rateRuleTestName\n# TYPE gauge \nrateRuleTestName 1.000000e+00\n",
		stringRate)
}
