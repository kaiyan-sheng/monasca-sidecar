// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateAvg(t *testing.T) {
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
	oldMetricFamilies, errParseOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errParseNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.NoError(t, errParseOldMF)
	assert.NoError(t, errParseNewMF)

	// define avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	// (30 + 25) / 2 = 27.5
	// (20 + 10) / 2 = 15
	avgMetricFamilies := calculateAvg(newMetricFamilies, oldMetricFamilies, avgRule)
	avgMetricString := convertMetricFamiliesIntoTextString(avgMetricFamilies)
	expectedAvgMetricString := `# HELP avgRuleTestName avgRuleTestName
# TYPE avgRuleTestName gauge
avgRuleTestName{method="GET",path="/rest/metrics"} 27.5
# HELP avgRuleTestName avgRuleTestName
# TYPE avgRuleTestName gauge
avgRuleTestName{method="POST",path="/rest/support"} 15
`
	assert.Equal(t, expectedAvgMetricString, avgMetricString)
}

func TestCalculateAvgWithMisMatchDimensions(t *testing.T) {
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

	// define avgRule
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}

	avgMetricFamilies := calculateAvg(newMetricFamilies, oldMetricFamilies, avgRule)
	assert.Equal(t, 0, len(avgMetricFamilies))
}

func TestFindOldValueWithHistogramAvg(t *testing.T) {
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
	assert.NoError(t, errOldMF)
	assert.NoError(t, errNewMF)
	newPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(newMetricFamilies)
	oldPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(oldMetricFamilies)

	// define avgRule
	avgRuleParam := map[string]string{}
	// avgRuleBucket
	avgRuleParam["name"] = "http_request_duration_seconds_bucket"
	avgRuleBucket := SidecarRule{Name: "avgRuleTestHistogramName", Function: "avg", Parameters: avgRuleParam}

	avgMetricFamiliesBucket := calculateAvg(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, avgRuleBucket)
	avgMetricStringBucket := convertMetricFamiliesIntoTextString(avgMetricFamiliesBucket)

	expectedResultBucket := `# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="+Inf"} 146820
# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="0.05"} 24554
# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="0.1"} 33944
# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="0.2"} 100892
# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="0.5"} 134389
# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName{le="1"} 134988
`
	assert.Equal(t, expectedResultBucket, avgMetricStringBucket)

	// avgRuleSum
	avgRuleParam["name"] = "http_request_duration_seconds_sum"
	avgRuleSum := SidecarRule{Name: "avgRuleTestHistogramName", Function: "avg", Parameters: avgRuleParam}

	avgMetricFamiliesSum := calculateAvg(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, avgRuleSum)
	avgMetricStringSum := convertMetricFamiliesIntoTextString(avgMetricFamiliesSum)

	expectedResultSum := `# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName 58423
`
	assert.Equal(t, expectedResultSum, avgMetricStringSum)

	// avgRuleCount
	avgRuleParam["name"] = "http_request_duration_seconds_count"
	avgRuleCount := SidecarRule{Name: "avgRuleTestHistogramName", Function: "avg", Parameters: avgRuleParam}

	avgMetricFamiliesCount := calculateAvg(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, avgRuleCount)
	avgMetricStringCount := convertMetricFamiliesIntoTextString(avgMetricFamiliesCount)

	expectedResultCount := `# HELP avgRuleTestHistogramName avgRuleTestHistogramName
# TYPE avgRuleTestHistogramName gauge
avgRuleTestHistogramName 146820
`
	assert.Equal(t, expectedResultCount, avgMetricStringCount)
}
