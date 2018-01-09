// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateRatio(t *testing.T) {
	prometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 30
request_count{method="POST",path="/rest/support"} 20
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.3
request_total_time{method="POST",path="/rest/support"} 0.5
`
	metricFamilies, _ := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	// 0.3 / 30 = 0.01
	// 0.5 / 20 = 0.025
	ratioMetricFamilies := calculateRatio(metricFamilies, ratioRule)
	ratioMetricString := convertMetricFamiliesIntoTextString(ratioMetricFamilies)
	expectedRatioMetricString := `# HELP ratioRuleTestName ratioRuleTestName
# TYPE ratioRuleTestName gauge
ratioRuleTestName{method="GET",path="/rest/metrics"} 0.01
# HELP ratioRuleTestName ratioRuleTestName
# TYPE ratioRuleTestName gauge
ratioRuleTestName{method="POST",path="/rest/support"} 0.025
`
	assert.Equal(t, expectedRatioMetricString, ratioMetricString)
}

func TestCalculateRatioWithMisMatchDimensions(t *testing.T) {
	prometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 25
request_count{method="POST",path="/rest/support/1"} 10
`
	metricFamilies, _ := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "request_total_time"
	ratioRuleParam["denominator"] = "request_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestName", Function: "ratio", Parameters: ratioRuleParam}

	ratioMetricFamilies := calculateRatio(metricFamilies, ratioRule)
	ratioMetricString := convertMetricFamiliesIntoTextString(ratioMetricFamilies)
	assert.Equal(t, "", ratioMetricString)
}

func TestFindOldValueWithHistogramRatio(t *testing.T) {
	prometheusMetricsString := `# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
http_request_duration_seconds_bucket{le="+Inf"} 200000
http_request_duration_seconds_sum 50000
http_request_duration_seconds_count 200000
`
	metricFamilies, _ := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	// define ratioRule
	// define ratioRule
	ratioRuleParam := map[string]string{}
	ratioRuleParam["numerator"] = "http_request_duration_seconds_sum"
	ratioRuleParam["denominator"] = "http_request_duration_seconds_count"
	ratioRule := SidecarRule{Name: "ratioRuleTestHistogramName", Function: "ratio", Parameters: ratioRuleParam}

	ratioMetricFamiliesBucket := calculateRatio(metricFamilies, ratioRule)
	ratioMetricStringBucket := convertMetricFamiliesIntoTextString(ratioMetricFamiliesBucket)

	// 50000 / 200000 = 0.25
	expectedResultBucket := `# HELP ratioRuleTestHistogramName ratioRuleTestHistogramName
# TYPE ratioRuleTestHistogramName gauge
ratioRuleTestHistogramName 0.25
`
	assert.Equal(t, expectedResultBucket, ratioMetricStringBucket)
}
