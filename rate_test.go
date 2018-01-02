// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateRate(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "2.0", Dimensions: newMetricDimension}
	oldValueString := "1"
	queryInterval := 10.0
	rateResult, errRate := calculateRate(newPrometheusMetric, oldValueString, queryInterval)
	// (2 - 1) / 10.0 = 0.1
	assert.Equal(t, 0.1, rateResult)
	assert.Equal(t, nil, errRate)
}

func TestCalculateRateNegative(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "1.0", Dimensions: newMetricDimension}
	oldValueString := "2"
	queryInterval := 10.0
	rateResult, errRate := calculateRate(newPrometheusMetric, oldValueString, queryInterval)
	// (1 - 2) / 10.0 = -0.1
	assert.Equal(t, -0.1, rateResult)
	assert.Equal(t, nil, errRate)
}

func TestCalculateRateWithBadValueString(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "abc", Dimensions: newMetricDimension}
	oldValueString := "1"
	queryInterval := 10.0
	rateResult, errRate := calculateRate(newPrometheusMetric, oldValueString, queryInterval)
	// Failed to convert "abc" to float64
	assert.Equal(t, 0.0, rateResult)
	assert.NotEqual(t, nil, errRate)
}

func TestStructNewStringRate(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "2.0", Dimensions: newMetricDimension}
	rateValue := 1.0
	stringRate := structNewStringRate(newPrometheusMetric, rateValue)
	assert.Equal(t,
		"# HELP test_calculate_rate_per_second\n# TYPE gauge \ntest_calculate_rate_per_second{} 1.000000e+00\n",
		stringRate)
}

func TestConvertDimensionsToString(t *testing.T) {
	dimension1 := Dimension{Key: "key1", Value: "value1"}
	dimension2 := Dimension{Key: "key2", Value: "value2"}
	dimensionList := DimensionList{dimension1, dimension2}
	dimensionString := dimensionsToString(dimensionList)
	assert.Equal(t, "{key1=value1,key2=value2,{key1=value1,key2=value2}", dimensionString)
}
