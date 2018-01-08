// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"
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
	metricDimensions1 := []Dimension{}
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key2", Value: "value2"})
	metricDimensions1 = append(metricDimensions1, Dimension{Key: "key1", Value: "value1"})
	dimensions1Hash := convertDimensionsToHash(metricDimensions1)

	metricDimensions2 := []Dimension{}
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key1", Value: "value1"})
	metricDimensions2 = append(metricDimensions2, Dimension{Key: "key2", Value: "value2"})
	dimensions2Hash := convertDimensionsToHash(metricDimensions2)
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

func TestRemoveDuplicates(t *testing.T) {
	elements := []string{"metric1", "metric2", "name1", "name2", "metric1", "metric2", "name1", "name2"}
	dedupElements := removeDuplicates(elements)
	expectedElements := []string{"metric1", "metric2", "name1", "name2"}
	assert.Equal(t, expectedElements, dedupElements)
}

func TestConvertDimensionsToString(t *testing.T) {
	dimension1 := Dimension{Key: "key1", Value: "value1"}
	dimension2 := Dimension{Key: "key2", Value: "value2"}
	dimensionList := DimensionList{dimension1, dimension2}
	dimensionString := dimensionsToString(dimensionList)
	assert.Equal(t, "{key1=value1,key2=value2}", dimensionString)
}

func TestStructNewStringRate(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_rate", Value: "2.0", Dimensions: newMetricDimension}
	rateValue := 1.0
	rateRuleParam := map[string]string{}
	rateRuleParam["name"] = "request_count"
	rateRule := SidecarRule{Name: "rateRuleTestName", Function: "rate", Parameters: rateRuleParam}
	stringRate := structNewMetricString(newPrometheusMetric, rateValue, rateRule)
	assert.Equal(t,
		"# HELP rateRuleTestName\n# TYPE gauge\nrateRuleTestName 1.000000e+00\n",
		stringRate)
}

func TestStructNewStringAvg(t *testing.T) {
	newMetricDimension := DimensionList{}
	newPrometheusMetric := PrometheusMetric{Name: "test_calculate_avg", Value: "2.0", Dimensions: newMetricDimension}
	avgValue := 1.0
	avgRuleParam := map[string]string{}
	avgRuleParam["name"] = "request_count"
	avgRule := SidecarRule{Name: "avgRuleTestName", Function: "avg", Parameters: avgRuleParam}
	stringAvg := structNewMetricString(newPrometheusMetric, avgValue, avgRule)
	assert.Equal(t,
		"# HELP avgRuleTestName\n# TYPE gauge\navgRuleTestName 1.000000e+00\n",
		stringAvg)
}

func TestFindDenominatorValue(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	numeratorDimHash := convertDimensionsToHash(metricDimensions)
	denominatorDimHash := convertDimensionsToHash(metricDimensionsDiff)

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: numeratorDimHash}
	prometheusMetric2 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensionsDiff, DimensionHash: denominatorDimHash}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)

	denominatorValue, errDenominator := findDenominatorValue(prometheusMetrics, numeratorDimHash, "request_count")
	assert.Equal(t, 2.0, denominatorValue)
	assert.Equal(t, nil, errDenominator)
}

func TestFindDenominatorValueFailed(t *testing.T) {
	metricDimensions := []Dimension{}
	metricDimensionsDiff := []Dimension{}
	prometheusMetrics := []PrometheusMetric{}

	// define dimensions
	metricDimensions = append(metricDimensions, Dimension{Key: "key1", Value: "value1"})
	metricDimensionsDiff = append(metricDimensionsDiff, Dimension{Key: "key3", Value: "value3"})
	numeratorDimHash := convertDimensionsToHash(metricDimensions)
	denominatorDimHash := convertDimensionsToHash(metricDimensionsDiff)

	// define prometheusMetrics
	prometheusMetric1 := PrometheusMetric{Name: "request_count", Value: "2.0", Dimensions: metricDimensions, DimensionHash: denominatorDimHash}
	prometheusMetric2 := PrometheusMetric{Name: "request_count", Value: "5.0", Dimensions: metricDimensionsDiff, DimensionHash: denominatorDimHash}
	prometheusMetrics = append(prometheusMetrics, prometheusMetric1)
	prometheusMetrics = append(prometheusMetrics, prometheusMetric2)

	denominatorValue, errDenominator := findDenominatorValue(prometheusMetrics, numeratorDimHash, "request_count")
	assert.Equal(t, 0.0, denominatorValue)
	assert.NotEqual(t, nil, errDenominator)
}

func TestParserTextToMetricFamilies(t *testing.T) {
	text := `
# HELP request_count Counts requests by method and path
# TYPE request_count counter
request_count{method="GET",path="/rest/metrics"} 25
`
	result, _ := parsePrometheusMetricsToMetricFamilies(text)
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
	results, _ := parsePrometheusMetricsToMetricFamilies(text)
	newResults := []*dto.MetricFamily{}
	for _, r := range results {
		if *r.Name != "request_count" {
			newResults = append(newResults, r)
		} else {
			for _, requestCountMetric := range r.Metric {
				requestCountLabels := requestCountMetric.Label
				for _, newRate := range createNewRatePrometheus("request_count_rate", requestCountLabels, 0.25) {
					newResults = append(newResults, newRate)
				}

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

func TestCalculateRateWithHistogram(t *testing.T) {
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
	//	newPrometheusMetricsString := `
	//	# A histogram, which has a pretty complex representation in the text format:
	//# HELP http_request_duration_seconds A histogram of the request duration.
	//# TYPE http_request_duration_seconds histogram
	//http_request_duration_seconds_bucket{le="0.05"} 25000
	//http_request_duration_seconds_bucket{le="0.1"} 34000
	//http_request_duration_seconds_bucket{le="0.2"} 101000
	//http_request_duration_seconds_bucket{le="0.5"} 130000
	//http_request_duration_seconds_bucket{le="1"} 135000
	//http_request_duration_seconds_bucket{le="+Inf"} 149000
	//http_request_duration_seconds_sum 60000
	//http_request_duration_seconds_count 149000
	//`
	histogramMetricFamilies, _ := parsePrometheusMetricsToMetricFamilies(histogramMetricsString)
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
