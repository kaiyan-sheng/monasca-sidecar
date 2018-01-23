// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestParseFloat(t *testing.T) {
	string1 := "30"
	float1, err1 := strconv.ParseFloat(string1, 64)
	assert.Equal(t, 30.0, float1)
	assert.NoError(t, err1)

	string2 := "30.0"
	float2, err2 := strconv.ParseFloat(string2, 64)
	assert.Equal(t, 30.0, float2)
	assert.NoError(t, err2)

	string3 := "not a float"
	float3, err3 := strconv.ParseFloat(string3, 64)
	assert.Equal(t, 0.0, float3)
	assert.Error(t, err3)

	string4 := "0"
	float4, err4 := strconv.ParseFloat(string4, 64)
	assert.Equal(t, 0.0, float4)
	assert.NoError(t, err4)

	string5 := "-30"
	float5, err5 := strconv.ParseFloat(string5, 64)
	assert.Equal(t, -30.0, float5)
	assert.NoError(t, err5)
}

func TestGetPrometheusUrlFromAnnotations(t *testing.T) {
	annotations1 := map[string]string{}
	annotations1["prometheus.io/scrape"] = "true"
	annotations1["sidecar/port"] = "5556"
	annotations1["sidecar/path"] = "/support/metrics"
	annotations1["sidecar/rules"] = ""
	prometheusUrl1, flag1 := getPrometheusUrl(annotations1)
	assert.True(t, flag1)
	assert.Equal(t, "http://localhost:5556/support/metrics", prometheusUrl1)

	// use default prometheus.io/path
	annotations2 := map[string]string{}
	annotations2["prometheus.io/scrape"] = "true"
	annotations2["sidecar/port"] = "5556"
	annotations2["sidecar/rules"] = ""
	prometheusUrl2, flag2 := getPrometheusUrl(annotations2)
	assert.True(t, flag2)
	assert.Equal(t, "http://localhost:5556/metrics", prometheusUrl2)

	// missing scrape=true
	annotations3 := map[string]string{}
	annotations3["prometheus.io/port"] = "9999"
	annotations3["prometheus.io/path"] = "/support/metrics"
	annotations3["sidecar/port"] = "5556"
	annotations3["sidecar/rules"] = ""
	prometheusUrl3, flag3 := getPrometheusUrl(annotations3)
	assert.False(t, flag3)
	assert.Equal(t, "", prometheusUrl3)
}
