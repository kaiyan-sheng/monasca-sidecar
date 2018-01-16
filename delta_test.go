// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateDelta(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
request_count{method="POST",path="/rest/support"} 10
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
request_total_time{method="POST",path="/rest/support"} 0.7
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 30
request_count{method="POST",path="/rest/support"} 20
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.9
request_total_time{method="POST",path="/rest/support"} 1.0
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define queryInterval and deltaRule
	deltaRuleParam := map[string]string{}
	deltaRuleParam["name"] = "request_count"
	deltaRule := SidecarRule{Name: "deltaRuleTestName", Function: "delta", Parameters: deltaRuleParam}

	// 30 - 25 = 5
	// 20 - 10 = 10
	deltaMetricFamilies := calculateDelta(newMetricFamilies, oldMetricFamilies, deltaRule)
	deltaMetricString := convertMetricFamiliesIntoTextString(deltaMetricFamilies)
	expectedDeltaMetricString := `# HELP deltaRuleTestName deltaRuleTestName
# TYPE deltaRuleTestName gauge
deltaRuleTestName{method="GET",path="/rest/metrics"} 5
# HELP deltaRuleTestName deltaRuleTestName
# TYPE deltaRuleTestName gauge
deltaRuleTestName{method="POST",path="/rest/support"} 10
`
	assert.Equal(t, expectedDeltaMetricString, deltaMetricString)
}

func TestCalculateDeltaWithMisMatchDimensions(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 25
request_count{method="POST",path="/rest/support/1"} 10
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/2"} 30
request_count{method="POST",path="/rest/support/2"} 20
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRule
	deltaRuleParam := map[string]string{}
	deltaRuleParam["name"] = "request_count"
	deltaRule := SidecarRule{Name: "deltaRuleTestName", Function: "delta", Parameters: deltaRuleParam}

	deltaMetricFamilies := calculateDelta(newMetricFamilies, oldMetricFamilies, deltaRule)
	assert.Equal(t, 0, len(deltaMetricFamilies))
}

func TestFindOldValueWithHistogramDelta(t *testing.T) {
	oldPrometheusMetricsString := `# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
http_request_duration_seconds_bucket{le="+Inf"} 144320
http_request_duration_seconds_sum 53423
http_request_duration_seconds_count 144320
`
	newPrometheusMetricsString := `# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_duration_seconds A histogram of the request duration.
# TYPE http_request_duration_seconds histogram
http_request_duration_seconds_bucket{le="0.05"} 25054
http_request_duration_seconds_bucket{le="0.1"} 34444
http_request_duration_seconds_bucket{le="0.2"} 101392
http_request_duration_seconds_bucket{le="0.5"} 139389
http_request_duration_seconds_bucket{le="1"} 135988
http_request_duration_seconds_bucket{le="+Inf"} 149320
http_request_duration_seconds_sum 63423
http_request_duration_seconds_count 149320
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	newPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(newMetricFamilies)
	oldPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(oldMetricFamilies)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRule
	deltaRuleParam := map[string]string{}
	// deltaRuleBucket
	deltaRuleParam["name"] = "http_request_duration_seconds_bucket"
	deltaRuleBucket := SidecarRule{Name: "deltaRuleTestHistogramName", Function: "delta", Parameters: deltaRuleParam}

	deltaMetricFamiliesBucket := calculateDelta(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, deltaRuleBucket)
	deltaMetricStringBucket := convertMetricFamiliesIntoTextString(deltaMetricFamiliesBucket)

	expectedResultBucket := `# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="+Inf"} 5000
# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="0.05"} 1000
# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="0.1"} 1000
# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="0.2"} 1000
# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="0.5"} 10000
# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName{le="1"} 2000
`
	assert.Equal(t, expectedResultBucket, deltaMetricStringBucket)

	// deltaRuleSum
	deltaRuleParam["name"] = "http_request_duration_seconds_sum"
	deltaRuleSum := SidecarRule{Name: "deltaRuleTestHistogramName", Function: "delta", Parameters: deltaRuleParam}

	deltaMetricFamiliesSum := calculateDelta(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, deltaRuleSum)
	deltaMetricStringSum := convertMetricFamiliesIntoTextString(deltaMetricFamiliesSum)

	expectedResultSum := `# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName 10000
`
	assert.Equal(t, expectedResultSum, deltaMetricStringSum)

	// deltaRuleCount
	deltaRuleParam["name"] = "http_request_duration_seconds_count"
	deltaRuleCount := SidecarRule{Name: "deltaRuleTestHistogramName", Function: "delta", Parameters: deltaRuleParam}

	deltaMetricFamiliesCount := calculateDelta(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, deltaRuleCount)
	deltaMetricStringCount := convertMetricFamiliesIntoTextString(deltaMetricFamiliesCount)

	expectedResultCount := `# HELP deltaRuleTestHistogramName deltaRuleTestHistogramName
# TYPE deltaRuleTestHistogramName gauge
deltaRuleTestHistogramName 5000
`
	assert.Equal(t, expectedResultCount, deltaMetricStringCount)
}

func TestCalculateDeltaWithResettingCounters(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 5
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define queryInterval and deltaRule
	deltaRuleParam := map[string]string{}
	deltaRuleParam["name"] = "request_count"
	deltaRule := SidecarRule{Name: "deltaRuleTestName", Function: "delta", Parameters: deltaRuleParam}

	deltaMetricFamilies := calculateDelta(newMetricFamilies, oldMetricFamilies, deltaRule)
	assert.Equal(t, 0, len(deltaMetricFamilies))
}
