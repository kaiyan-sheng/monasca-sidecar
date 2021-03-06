// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateDeltaRatio(t *testing.T) {
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
request_total_time{method="POST",path="/rest/support"} 1.2
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.9 - 0.5) / (30 - 25) = 0.08
	// (1.2 - 0.7) / (20 - 10) = 0.05
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)
	expectedDeltaRatioMetricString := `# HELP deltaRatioRuleTestName deltaRatioRuleTestName
# TYPE deltaRatioRuleTestName gauge
deltaRatioRuleTestName{method="GET",path="/rest/metrics"} 0.08
# HELP deltaRatioRuleTestName deltaRatioRuleTestName
# TYPE deltaRatioRuleTestName gauge
deltaRatioRuleTestName{method="POST",path="/rest/support"} 0.05
`
	assert.Equal(t, expectedDeltaRatioMetricString, deltaRatioMetricString)
}

func TestCalculateDeltaRatioWithMisMatchDimensions(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 25
request_count{method="POST",path="/rest/support/1"} 10
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics/1"} 0.5
request_total_time{method="POST",path="/rest/support/1"} 0.7
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/2"} 30
request_count{method="POST",path="/rest/support/2"} 20
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics/2"} 0.9
request_total_time{method="POST",path="/rest/support/2"} 1.0
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// mismatch dimensions
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	assert.Equal(t, 0, len(deltaRatioMetricFamilies))
}

func TestFindOldValueWithHistogramDeltaRatio(t *testing.T) {
	oldPrometheusMetricsString := `# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_dudeltaRation_seconds A histogram of the request dudeltaRation.
# TYPE http_request_dudeltaRation_seconds histogram
http_request_dudeltaRation_seconds_bucket{le="0.05"} 24054
http_request_dudeltaRation_seconds_bucket{le="0.1"} 33444
http_request_dudeltaRation_seconds_bucket{le="0.2"} 100392
http_request_dudeltaRation_seconds_bucket{le="0.5"} 129389
http_request_dudeltaRation_seconds_bucket{le="1"} 133988
http_request_dudeltaRation_seconds_bucket{le="+Inf"} 144320
http_request_dudeltaRation_seconds_sum 53423
http_request_dudeltaRation_seconds_count 144320
`
	newPrometheusMetricsString := `# A histogram, which has a pretty complex representation in the text format:
# HELP http_request_dudeltaRation_seconds A histogram of the request dudeltaRation.
# TYPE http_request_dudeltaRation_seconds histogram
http_request_dudeltaRation_seconds_bucket{le="0.05"} 25054
http_request_dudeltaRation_seconds_bucket{le="0.1"} 34444
http_request_dudeltaRation_seconds_bucket{le="0.2"} 101392
http_request_dudeltaRation_seconds_bucket{le="0.5"} 139389
http_request_dudeltaRation_seconds_bucket{le="1"} 135988
http_request_dudeltaRation_seconds_bucket{le="+Inf"} 149320
http_request_dudeltaRation_seconds_sum 63423
http_request_dudeltaRation_seconds_count 149320
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)
	newPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(newMetricFamilies)
	oldPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(oldMetricFamilies)

	// define deltaRatioRule
	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "http_request_dudeltaRation_seconds_count"
	deltaRatioRuleParam["denominator"] = "http_request_dudeltaRation_seconds_sum"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestHistogramName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	deltaRatioMetricFamilies := calculateDeltaRatio(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)

	// (149320 - 144320) / (63423 - 53423) = 0.5
	expectedResult := `# HELP deltaRatioRuleTestHistogramName deltaRatioRuleTestHistogramName
# TYPE deltaRatioRuleTestHistogramName gauge
deltaRatioRuleTestHistogramName 0.5
`
	assert.Equal(t, expectedResult, deltaRatioMetricString)
}

func TestCalculateDeltaRatioInf(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.6
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.6 - 0.5) / (25 - 25) = +Inf
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)
	assert.Equal(t, "", deltaRatioMetricString)
}

func TestCalculateDeltaRatioZero(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 24
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.5 - 0.5) / (25 - 24) = 0
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)
	expectedDeltaRatioMetricString := `# HELP deltaRatioRuleTestName deltaRatioRuleTestName
# TYPE deltaRatioRuleTestName gauge
deltaRatioRuleTestName{method="GET",path="/rest/metrics"} 0
`
	assert.Equal(t, expectedDeltaRatioMetricString, deltaRatioMetricString)
}

func TestCalculateDeltaRatioNaN(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// (0.5 - 0.5) / (25 - 25) = NaN
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)
	assert.Equal(t, "", deltaRatioMetricString)
}

func TestCalculateDeltaRatioWithResettingCounters(t *testing.T) {
	oldPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.5
`
	newPrometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 5
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.1
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_total_time"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "deltaRatioRuleTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// test resetting counters
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	assert.Equal(t, 0, len(deltaRatioMetricFamilies))
}

func TestDeltaRatioWithRequstBucketCount(t *testing.T) {
	oldPrometheusMetricsString := `# HELP request_bucket_count Histogram count of requests by method, path (seconds), bucket
# TYPE request_bucket_count counter
request_bucket_count{ge=".2",method="GET",path="/rest/appliances"} 5
request_bucket_count{ge=".5",method="GET",path="/rest/appliances"} 1
request_bucket_count{ge="1",method="GET",path="/rest/appliances"} 1
request_bucket_count{ge="2.5",method="GET",path="/rest/metrics"} 2
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/appliances"} 7
request_count{method="GET",path="/rest/metrics"} 2
`
	newPrometheusMetricsString := `# HELP request_bucket_count Histogram count of requests by method, path (seconds), bucket
# TYPE request_bucket_count counter
request_bucket_count{ge=".2",method="GET",path="/rest/appliances"} 6
request_bucket_count{ge=".5",method="GET",path="/rest/appliances"} 3
request_bucket_count{ge="1",method="GET",path="/rest/appliances"} 1
request_bucket_count{ge="1",method="GET",path="/rest/metrics"} 1
request_bucket_count{ge="2.5",method="GET",path="/rest/metrics"} 3
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/appliances"} 10
request_count{method="GET",path="/rest/metrics"} 4
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)

	// define deltaRatioRule
	deltaRatioRuleParam := map[string]string{}
	deltaRatioRuleParam["numerator"] = "request_bucket_count"
	deltaRatioRuleParam["denominator"] = "request_count"
	deltaRatioRule := SidecarRule{Name: "requestBucketCountRatioTestName", Function: "deltaRatio", Parameters: deltaRatioRuleParam}

	// delta ratio
	deltaRatioMetricFamilies := calculateDeltaRatio(newMetricFamilies, oldMetricFamilies, deltaRatioRule)
	deltaRatioMetricString := convertMetricFamiliesIntoTextString(deltaRatioMetricFamilies)
	expectedResult := `# HELP requestBucketCountRatioTestName requestBucketCountRatioTestName
# TYPE requestBucketCountRatioTestName gauge
requestBucketCountRatioTestName{ge=".2",method="GET",path="/rest/appliances"} 0.3333333333333333
# HELP requestBucketCountRatioTestName requestBucketCountRatioTestName
# TYPE requestBucketCountRatioTestName gauge
requestBucketCountRatioTestName{ge=".5",method="GET",path="/rest/appliances"} 0.6666666666666666
# HELP requestBucketCountRatioTestName requestBucketCountRatioTestName
# TYPE requestBucketCountRatioTestName gauge
requestBucketCountRatioTestName{ge="1",method="GET",path="/rest/appliances"} 0
# HELP requestBucketCountRatioTestName requestBucketCountRatioTestName
# TYPE requestBucketCountRatioTestName gauge
requestBucketCountRatioTestName{ge="2.5",method="GET",path="/rest/metrics"} 0.5
`
	assert.Equal(t, expectedResult, deltaRatioMetricString)
}
