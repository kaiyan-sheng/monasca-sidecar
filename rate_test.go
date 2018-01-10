// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFindOldValueWithMetricFamilyRate(t *testing.T) {
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
`
	oldMetricFamilies, errOldMF := parsePrometheusMetricsToMetricFamilies(oldPrometheusMetricsString)
	newMetricFamilies, errNewMF := parsePrometheusMetricsToMetricFamilies(newPrometheusMetricsString)
	assert.Equal(t, nil, errOldMF)
	assert.Equal(t, nil, errNewMF)
	for _, newMF := range newMetricFamilies {
		for _, newMetric := range newMF.Metric {
			oldValueString, oldValueFloat := findOldValueWithMetricFamily(oldMetricFamilies, newMetric, *newMF.Name, *newMF.Type)
			assert.Equal(t, oldValueString, "value:25 ")
			assert.Equal(t, oldValueFloat, 25.0)
		}
	}
}

func TestCalculateRate(t *testing.T) {
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
	assert.Equal(t, nil, errNewMF)
	assert.Equal(t, nil, errOldMF)

	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (30 - 25) / 10.0 = 0.5
	// (20 - 10) / 10.0 = 1.0
	rateMetricFamilies := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRule)
	rateMetricString := convertMetricFamiliesIntoTextString(rateMetricFamilies)
	expectedRateMetricString := `# HELP rateRuleTestName rateRuleTestName
# TYPE rateRuleTestName gauge
rateRuleTestName{method="GET",path="/rest/metrics"} 0.5
# HELP rateRuleTestName rateRuleTestName
# TYPE rateRuleTestName gauge
rateRuleTestName{method="POST",path="/rest/support"} 1
`
	assert.Equal(t, expectedRateMetricString, rateMetricString)
}

func TestCalculateRateWithMisMatchDimensions(t *testing.T) {
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
	assert.Equal(t, nil, errNewMF)
	assert.Equal(t, nil, errOldMF)

	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	rateMetricFamilies := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRule)
	rateMetricString := convertMetricFamiliesIntoTextString(rateMetricFamilies)
	assert.Equal(t, "", rateMetricString)
}

func TestFindOldValueWithHistogramRate(t *testing.T) {
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
	assert.Equal(t, nil, errNewMF)
	assert.Equal(t, nil, errOldMF)

	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	// rateRuleBucket
	rateRuleParam["name"] = "http_request_duration_seconds_bucket"
	rateRuleBucket := SidecarRule{Name: "rateRuleTestHistogramName", Function: "rate", Parameters: rateRuleParam}

	rateMetricFamiliesBucket := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRuleBucket)
	rateMetricStringBucket := convertMetricFamiliesIntoTextString(rateMetricFamiliesBucket)

	expectedResultBucket := `# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="+Inf"} 500
# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="0.05"} 100
# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="0.1"} 100
# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="0.2"} 100
# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="0.5"} 1000
# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName{le="1"} 200
`
	assert.Equal(t, expectedResultBucket, rateMetricStringBucket)

	// rateRuleSum
	rateRuleParam["name"] = "http_request_duration_seconds_sum"
	rateRuleSum := SidecarRule{Name: "rateRuleTestHistogramName", Function: "rate", Parameters: rateRuleParam}

	rateMetricFamiliesSum := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRuleSum)
	rateMetricStringSum := convertMetricFamiliesIntoTextString(rateMetricFamiliesSum)

	expectedResultSum := `# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName 1000
`
	assert.Equal(t, expectedResultSum, rateMetricStringSum)

	// rateRuleCount
	rateRuleParam["name"] = "http_request_duration_seconds_count"
	rateRuleCount := SidecarRule{Name: "rateRuleTestHistogramName", Function: "rate", Parameters: rateRuleParam}

	rateMetricFamiliesCount := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRuleCount)
	rateMetricStringCount := convertMetricFamiliesIntoTextString(rateMetricFamiliesCount)

	expectedResultCount := `# HELP rateRuleTestHistogramName rateRuleTestHistogramName
# TYPE rateRuleTestHistogramName gauge
rateRuleTestHistogramName 500
`
	assert.Equal(t, expectedResultCount, rateMetricStringCount)
}

func TestCalculateRateWithResettingCounters(t *testing.T) {
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
	assert.Equal(t, nil, errNewMF)
	assert.Equal(t, nil, errOldMF)

	// define queryInterval and rateRule
	queryInterval := 10.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}

	// (30 - 25) / 10.0 = 0.5
	// (20 - 10) / 10.0 = 1.0
	rateMetricFamilies := calculateRate(newMetricFamilies, oldMetricFamilies, queryInterval, rateRule)
	rateMetricString := convertMetricFamiliesIntoTextString(rateMetricFamilies)
	assert.Equal(t, "", rateMetricString)
}
