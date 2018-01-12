// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSplitRules(t *testing.T) {
	var rules = `
- metricName: request_time_count_ratio
  function: ratio
  parameters:
    numerator: request_total_time
    denominator: request_count
- metricName: request_time_avg
  function: avg
  parameters:
    name: request_total_time
- metricName: request_count_rate
  function: rate
  parameters:
    name: request_count
- metricName: request_delta_ratio
  function: ratio
  parameters:
    numerator: request_total_time
    denominator: request_count`

	ruleStruct := parseYamlSidecarRules(rules)
	var expectedRules []SidecarRule
	param1 := map[string]string{}
	param1["numerator"] = "request_total_time"
	param1["denominator"] = "request_count"

	param2 := map[string]string{}
	param2["name"] = "request_total_time"

	param3 := map[string]string{}
	param3["name"] = "request_count"
	expectedRules = append(expectedRules, SidecarRule{Name: "request_time_count_ratio", Function: "ratio", Parameters: param1})
	expectedRules = append(expectedRules, SidecarRule{Name: "request_time_avg", Function: "avg", Parameters: param2})
	expectedRules = append(expectedRules, SidecarRule{Name: "request_count_rate", Function: "rate", Parameters: param3})
	expectedRules = append(expectedRules, SidecarRule{Name: "request_delta_ratio", Function: "ratio", Parameters: param1})
	assert.Equal(t, expectedRules, ruleStruct)
}

func TestFindDenominatorValue(t *testing.T) {
	prometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 30
request_count{method="POST",path="/rest/support"} 20
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics"} 0.9
request_total_time{method="POST",path="/rest/support"} 1.2
`
	labelPairs1 := []*dto.LabelPair{
		{Name: proto.String("method"), Value: proto.String("GET")},
		{Name: proto.String("path"), Value: proto.String("/rest/metrics")},
	}
	labelPairs2 := []*dto.LabelPair{
		{Name: proto.String("method"), Value: proto.String("POST")},
		{Name: proto.String("path"), Value: proto.String("/rest/support")},
	}
	metricFamilies, err := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	assert.NoError(t, err)
	for _, metricFamily := range metricFamilies {
		for _, m := range metricFamily.Metric {
			newDenominatorValueFloat, succeedNewDenominator := findDenominatorValue(metricFamilies, m.Label, "request_total_time")
			if checkEqualLabels(m.Label, labelPairs1) {
				assert.True(t, succeedNewDenominator)
				assert.Equal(t, 0.9, newDenominatorValueFloat)
			}
			if checkEqualLabels(m.Label, labelPairs2) {
				assert.True(t, succeedNewDenominator)
				assert.Equal(t, 1.2, newDenominatorValueFloat)
			}
		}
	}

}

func TestFindDenominatorValueMisMatchLabels(t *testing.T) {
	prometheusMetricsString := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 30
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics/1"} 0.9
`
	labelPairs := []*dto.LabelPair{
		{Name: proto.String("method"), Value: proto.String("GET")},
		{Name: proto.String("path"), Value: proto.String("/rest/metrics")},
	}
	metricFamilies, err := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	assert.NoError(t, err)
	newDenominatorValueFloat, succeedNewDenominator := findDenominatorValue(metricFamilies, labelPairs, "request_total_time")
	assert.False(t, succeedNewDenominator)
	assert.Equal(t, 0.0, newDenominatorValueFloat)
}

func TestParserTextToMetricFamilies(t *testing.T) {
	text := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
`
	result, err := parsePrometheusMetricsToMetricFamilies(text)
	assert.NoError(t, err)
	expectLabelPairs := []*dto.LabelPair{
		{Name: proto.String("method"), Value: proto.String("GET")},
		{Name: proto.String("path"), Value: proto.String("/rest/metrics")},
	}

	for _, r := range result {
		assert.Equal(t, "COUNTER", r.Type.String())
		assert.Equal(t, "request_count", *r.Name)
		metric := r.Metric
		for _, m := range metric {
			assert.Equal(t, "value:25 ", m.Counter.String())
			assert.Equal(t, 25.0, m.Counter.GetValue())
			assert.Equal(t, "<nil>", m.Gauge.String())
			assert.Equal(t, "<nil>", m.Histogram.String())
			assert.Equal(t, "<nil>", m.Summary.String())
			assert.Equal(t, expectLabelPairs, m.Label)
		}
	}
}

func TestConvertMetricFamiliesToText(t *testing.T) {
	text := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 1027 1395066363000
http_requests_total{method="post",code="400"}    3 1395066363000
`
	results, err := parsePrometheusMetricsToMetricFamilies(text)
	assert.NoError(t, err)
	newResults := []*dto.MetricFamily{}
	for _, r := range results {
		if *r.Name != "request_count" {
			newResults = append(newResults, r)
		} else {
			for _, requestCountMetric := range r.Metric {
				requestCountLabels := requestCountMetric.Label
				newResults = append(newResults, createNewMetricFamilies("request_count_rate", requestCountLabels, 0.25))
			}
		}
	}
	newResultsString := convertMetricFamiliesIntoTextString(newResults)

	expectedNewString := `# HELP request_count_rate request_count_rate
# TYPE request_count_rate gauge
request_count_rate{method="GET",path="/rest/metrics"} 0.25
# HELP http_requests_total The total number of HTTP requests.
# TYPE http_requests_total counter
http_requests_total{method="post",code="200"} 1027 1395066363000
http_requests_total{method="post",code="400"} 3 1395066363000
`
	assert.Equal(t, expectedNewString, newResultsString)
}

func TestConvertHistogramToGauge(t *testing.T) {
	histogramMetricsString := `# A histogram, which has a pretty complex representation in the text format:
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
	histogramMetricFamilies, err := parsePrometheusMetricsToMetricFamilies(histogramMetricsString)
	assert.NoError(t, err)
	convertedHistogramMetricFamilies := convertHistogramToGauge(histogramMetricFamilies[0])
	convertHistogramToGaugeString := convertMetricFamiliesIntoTextString(convertedHistogramMetricFamilies)
	expectedString := `# HELP http_request_duration_seconds_bucket A histogram of the request duration.
# TYPE http_request_duration_seconds_bucket gauge
http_request_duration_seconds_bucket{le="+Inf"} 144320
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
# HELP http_request_duration_seconds_count A histogram of the request duration.
# TYPE http_request_duration_seconds_count gauge
http_request_duration_seconds_count 144320
# HELP http_request_duration_seconds_sum A histogram of the request duration.
# TYPE http_request_duration_seconds_sum gauge
http_request_duration_seconds_sum 53423
`
	assert.Equal(t, expectedString, convertHistogramToGaugeString)
}

func TestConvertSummaryToGauge(t *testing.T) {
	summaryMetricsString := `# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 4.8738e-05
go_gc_duration_seconds{quantile="0.25"} 9.3497e-05
go_gc_duration_seconds{quantile="0.5"} 0.000374365
go_gc_duration_seconds{quantile="0.75"} 0.008759014
go_gc_duration_seconds{quantile="1"} 0.187098416
go_gc_duration_seconds_sum 1.289634876
go_gc_duration_seconds_count 49
`
	summaryMetricFamilies, err := parsePrometheusMetricsToMetricFamilies(summaryMetricsString)
	assert.NoError(t, err)
	convertedSummaryMetricFamilies := convertSummaryToGauge(summaryMetricFamilies[0])
	convertSummaryToGaugeString := convertMetricFamiliesIntoTextString(convertedSummaryMetricFamilies)
	expectedString := `# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds gauge
go_gc_duration_seconds{quantile="0"} 4.8738e-05
go_gc_duration_seconds{quantile="0.25"} 9.3497e-05
go_gc_duration_seconds{quantile="0.5"} 0.000374365
go_gc_duration_seconds{quantile="0.75"} 0.008759014
go_gc_duration_seconds{quantile="1"} 0.187098416
# HELP go_gc_duration_seconds_count A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds_count gauge
go_gc_duration_seconds_count 49
# HELP go_gc_duration_seconds_sum A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds_sum gauge
go_gc_duration_seconds_sum 1.289634876
`
	assert.Equal(t, expectedString, convertSummaryToGaugeString)
}

func TestReplaceHistogramSummaryToGauge(t *testing.T) {
	prometheusMetricsString := `# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 4.8738e-05
go_gc_duration_seconds{quantile="0.25"} 9.3497e-05
go_gc_duration_seconds{quantile="0.5"} 0.000374365
go_gc_duration_seconds{quantile="0.75"} 0.008759014
go_gc_duration_seconds{quantile="1"} 0.187098416
go_gc_duration_seconds_sum 1.289634876
go_gc_duration_seconds_count 49
# A histogram, which has a pretty complex representation in the text format:
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
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 30
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics/1"} 0.9
`
	prometheusMetricFamilies, err := parsePrometheusMetricsToMetricFamilies(prometheusMetricsString)
	assert.NoError(t, err)
	replacedMetricFamilies := replaceHistogramSummaryToGauge(prometheusMetricFamilies)
	replacedMetricFamiliesString := convertMetricFamiliesIntoTextString(replacedMetricFamilies)
	expectedString := `# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds gauge
go_gc_duration_seconds{quantile="0"} 4.8738e-05
go_gc_duration_seconds{quantile="0.25"} 9.3497e-05
go_gc_duration_seconds{quantile="0.5"} 0.000374365
go_gc_duration_seconds{quantile="0.75"} 0.008759014
go_gc_duration_seconds{quantile="1"} 0.187098416
# HELP go_gc_duration_seconds_count A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds_count gauge
go_gc_duration_seconds_count 49
# HELP go_gc_duration_seconds_sum A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds_sum gauge
go_gc_duration_seconds_sum 1.289634876
# HELP http_request_duration_seconds_bucket A histogram of the request duration.
# TYPE http_request_duration_seconds_bucket gauge
http_request_duration_seconds_bucket{le="+Inf"} 144320
http_request_duration_seconds_bucket{le="0.05"} 24054
http_request_duration_seconds_bucket{le="0.1"} 33444
http_request_duration_seconds_bucket{le="0.2"} 100392
http_request_duration_seconds_bucket{le="0.5"} 129389
http_request_duration_seconds_bucket{le="1"} 133988
# HELP http_request_duration_seconds_count A histogram of the request duration.
# TYPE http_request_duration_seconds_count gauge
http_request_duration_seconds_count 144320
# HELP http_request_duration_seconds_sum A histogram of the request duration.
# TYPE http_request_duration_seconds_sum gauge
http_request_duration_seconds_sum 53423
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics/1"} 30
# HELP request_total_time Total time in second requests take by method and path
# TYPE request_total_time counter
request_total_time{method="GET",path="/rest/metrics/1"} 0.9
`
	assert.Equal(t, expectedString, replacedMetricFamiliesString)
}
