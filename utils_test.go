// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

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
	dimensions1 := map[string]string{}
	dimensions1["key1"] = "value1"
	dimensions1["key2"] = "value2"
	dimensions1Hash := convertDimensionsToHash(dimensions1)

	dimensions2 := map[string]string{}
	dimensions2["key2"] = "value2"
	dimensions2["key1"] = "value1"
	dimensions2Hash := convertDimensionsToHash(dimensions2)
	assert.NotEqual(t, dimensions1Hash, dimensions2Hash)
}

func TestSortDimensionsByKeys(t *testing.T) {
	dimensions1 := map[string]string{}
	dimensions1["key3"] = "value3"
	dimensions1["key1"] = "value1"
	dimensions1["key2"] = "value2"
	dimensions1Sorted := sortDimensionsByKeys(dimensions1)

	expectedResult1 := map[string]string{}
	expectedResult1["key1"] = "value1"
	expectedResult1["key2"] = "value2"
	expectedResult1["key3"] = "value3"
	assert.Equal(t, expectedResult1, dimensions1Sorted)

	dimensions2 := map[string]string{}
	dimensions2["bc"] = "2"
	dimensions2["ab"] = "1"
	dimensions2["cd"] = "3"
	dimensions2Sorted := sortDimensionsByKeys(dimensions2)

	expectedResult2 := map[string]string{}
	expectedResult2["ab"] = "1"
	expectedResult2["bc"] = "2"
	expectedResult2["cd"] = "3"
	assert.Equal(t, expectedResult2, dimensions2Sorted)

	dimensions3 := map[string]string{}
	dimensions3["3"] = "3"
	dimensions3["2"] = "2"
	dimensions3["1"] = "1"
	dimensions3Sorted := sortDimensionsByKeys(dimensions3)

	expectedResult3 := map[string]string{}
	expectedResult3["1"] = "1"
	expectedResult3["2"] = "2"
	expectedResult3["3"] = "3"
	assert.Equal(t, expectedResult3, dimensions3Sorted)
}
