// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetSubString(t *testing.T) {
	testString := `request_count{method="GET",path="/rest/providers"} 1`
	start := "{"
	end := "}"
	subString := stringBetween(testString, start, end)
	assert.Equal(t, "method=\"GET\",path=\"/rest/providers\"", subString)
}

func TestGetSubStringWithWrongStartByte(t *testing.T) {
	testString := `abcd=1234`
	start := "x"
	end := "="
	subString := stringBetween(testString, start, end)
	assert.Equal(t, "", subString, "Start byte does not exist in original string")
}

func TestGetSubStringWithWrongEndByte(t *testing.T) {
	testString := `abcd=1234`
	start := "x"
	end := "="
	subString := stringBetween(testString, start, end)
	assert.Equal(t, "", subString, "End byte does not exist in original string")
}

func TestGetSubStringWithWrongStartEndByte(t *testing.T) {
	testString := `abcd=1234`
	start := "x"
	end := "y"
	subString := stringBetween(testString, start, end)
	assert.Equal(t, "", subString, "Start and end byte does not exist in original string")
}

func TestGetSubStringWithChars(t *testing.T) {
	testString := `request_count{method="GET"} 1`
	start := "{method="
	end := "} 1"
	subString := stringBetween(testString, start, end)
	assert.Equal(t, "\"GET\"", subString)
}

func TestGetSubStringWithDuplicateChars(t *testing.T) {
	testString1 := `aefd!=abcd`
	start1 := "a"
	end1 := "d"
	subString1 := stringBetween(testString1, start1, end1)
	assert.Equal(t, "ef", subString1)

	testString2 := `abcd!=aefd`
	start2 := "a"
	end2 := "d"
	subString2 := stringBetween(testString2, start2, end2)
	assert.Equal(t, "bc", subString2)
}

func TestGetPrometheusUrl(t *testing.T) {
	prometheusPort := "5556"
	prometheusPath1 := "/"
	url1 := getPrometheusUrl(prometheusPort, prometheusPath1)
	assert.Equal(t, "http://localhost:5556", url1)

	prometheusPath2 := "/metrics"
	url2 := getPrometheusUrl(prometheusPort, prometheusPath2)
	assert.Equal(t, "http://localhost:5556/metrics", url2)

	prometheusPath3 := "/metrics/"
	url3 := getPrometheusUrl(prometheusPort, prometheusPath3)
	assert.Equal(t, "http://localhost:5556/metrics", url3)

	prometheusPath4 := "/support/metrics"
	url4 := getPrometheusUrl(prometheusPort, prometheusPath4)
	assert.Equal(t, "http://localhost:5556/support/metrics", url4)

	prometheusPath5 := "/support/metrics/"
	url5 := getPrometheusUrl(prometheusPort, prometheusPath5)
	assert.Equal(t, "http://localhost:5556/support/metrics", url5)
}

func TestConvertDimensionsToHash(t *testing.T) {
	metricDimensions1 := DimensionList{}
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key2", Value: "value2"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key1", Value: "value1"})
	dimensions1Hash := convertDimensionsToHash(metricDimensions1)

	metricDimensions2 := DimensionList{}
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key1", Value: "value1"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key2", Value: "value2"})
	dimensions2Hash := convertDimensionsToHash(metricDimensions2)
	assert.NotEqual(t, dimensions1Hash, dimensions2Hash)
}

func TestSortDimensionsByKeys(t *testing.T) {
	metricDimensions1 := DimensionList{}
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key2", Value: "value2"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "a", Value: "b"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key1", Value: "value1"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "path", Value: "pathValue"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "1", Value: "2"})

	sortedMetricDimensions1 := sortDimensionsByKeys(metricDimensions1)

	expectedResult1 := DimensionList{}
	expectedResult1 = append(expectedResult1, Dimension{Key: "1", Value: "2"})
	expectedResult1 = append(expectedResult1, Dimension{Key: "a", Value: "b"})
	expectedResult1 = append(expectedResult1, Dimension{Key: "key1", Value: "value1"})
	expectedResult1 = append(expectedResult1, Dimension{Key: "key2", Value: "value2"})
	expectedResult1 = append(expectedResult1, Dimension{Key: "path", Value: "pathValue"})
	assert.Equal(t, expectedResult1, sortedMetricDimensions1)

	// test empty dimension list
	metricDimensions2 := DimensionList{}
	sortedMetricDimensions2 := sortDimensionsByKeys(metricDimensions2)
	expectedResult2 := DimensionList{}
	assert.Equal(t, expectedResult2, sortedMetricDimensions2)
}
