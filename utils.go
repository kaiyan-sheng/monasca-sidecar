// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.hpe.com/kronos/kelog"
	"gopkg.in/yaml.v2"
	"strconv"
	"strings"
)

type SidecarRule struct {
	Name       string            `yaml:"metricName"`
	Function   string            `yaml:"function"`
	Parameters map[string]string `yaml:"parameters"`
}

func stringBetween(value string, a string, b string) string {
	// Get substring between two strings.
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		log.Warnf("Start chars do not exist in original string")
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		log.Warnf("End chars do not exist in original string")
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	if posFirstAdjusted >= posLast {
		log.Warnf("Start chars is on the right side of end chars")
		return ""
	}
	return value[posFirstAdjusted:posLast]
}

func getSidecarRulesFromAnnotations(annotations map[string]string) (string, float64, string) {
	//get sidecar specific input parameters
	queryIntervalString := annotations["sidecar/query-interval"]
	if queryIntervalString == "" {
		log.Fatalf("sidecar/query-interval can not be empty")
	}

	listenPort := annotations["sidecar/listen-port"]
	if queryIntervalString == "" {
		log.Fatalf("sidecar/listenPort can not be empty")
	}

	queryInterval, errParseFloat := strconv.ParseFloat(queryIntervalString, 64)
	if queryInterval <= 0.0 || errParseFloat != nil {
		log.Warnf("Error converting \"sidecar/query-interval\": %v. Set queryInterval to default 30.0 seconds.", errParseFloat)
		queryInterval = 30.0
	}

	rules := annotations["sidecar/rules"]
	if rules == "" {
		log.Fatalf("sidecar/rules can not be empty")
	}
	log.Infof("rules = %s\n", rules)
	return rules, queryInterval, listenPort
}

func parseYamlSidecarRules(rules string) []SidecarRule {
	var ruleStruct []SidecarRule
	source := []byte(rules)
	err := yaml.Unmarshal(source, &ruleStruct)
	if err != nil {
		log.Fatalf("Error parsing sidecar rules: ", err)
	}
	return ruleStruct
}

func findDenominatorValue(prometheusMetrics []*dto.MetricFamily, numeratorLabels []*dto.LabelPair, denominatorName string) (float64, bool) {
	for _, pm := range prometheusMetrics {
		if *pm.Name == denominatorName {
			for _, metric := range pm.Metric {
				if checkEqualLabels(numeratorLabels, metric.Label) {
					denominatorValueFloat, succeedGetDenominator := getValueBasedOnType(*pm.Type, *metric)
					return denominatorValueFloat, succeedGetDenominator
				}
			}
		}
	}
	return 0.0, false
}

func checkEqualLabels(a, b []*dto.LabelPair) bool {
	if a == nil && b == nil {
		return true
	}

	if a == nil || b == nil {
		return false
	}

	if len(a) != len(b) {
		return false
	}
	for i, subA := range a {
		if (*subA.Name) != *b[i].Name || *subA.Value != *b[i].Value {
			return false
		}
	}
	return true
}

func parsePrometheusMetricsToMetricFamilies(text string) ([]*dto.MetricFamily, error) {
	var parser expfmt.TextParser
	parsed, err := parser.TextToMetricFamilies(strings.NewReader(text))
	if err != nil {
		return nil, err
	}
	var result []*dto.MetricFamily
	for _, mf := range parsed {
		result = append(result, mf)
	}
	return result, nil
}

func convertMetricFamiliesIntoTextString(newMetricFamilies []*dto.MetricFamily) string {
	// convert new metric families into text
	out := &bytes.Buffer{}
	for _, newMF := range newMetricFamilies {
		expfmt.MetricFamilyToText(out, newMF)
	}
	return out.String()
}

func convertHistogramToGauge(histogramMetricFamilies *dto.MetricFamily) []*dto.MetricFamily {
	reg := prometheus.NewRegistry()
	histogramBucketMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: *histogramMetricFamilies.Name + "_bucket",
			Help: *histogramMetricFamilies.Help,
		},
		[]string{
			"le",
		},
	)
	histogramSumMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: *histogramMetricFamilies.Name + "_sum",
			Help: *histogramMetricFamilies.Help,
		},
	)
	histogramCountMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: *histogramMetricFamilies.Name + "_count",
			Help: *histogramMetricFamilies.Help,
		},
	)
	reg.MustRegister(histogramBucketMetric)
	reg.MustRegister(histogramSumMetric)
	reg.MustRegister(histogramCountMetric)
	for _, histogramMetric := range histogramMetricFamilies.Metric {
		histogramSumValue := float64(*histogramMetric.Histogram.SampleSum)
		histogramSumMetric.Set(histogramSumValue)
		histogramCountValue := float64(*histogramMetric.Histogram.SampleCount)
		histogramCountMetric.Set(histogramCountValue)
		histogramBuckets := histogramMetric.Histogram.Bucket
		for _, hBucket := range histogramBuckets {
			histogramValue := float64(*hBucket.CumulativeCount)
			labelValue := strconv.FormatFloat(*hBucket.UpperBound, 'f', -1, 64)
			histogramBucketMetric.WithLabelValues(labelValue).Set(histogramValue)
		}
	}

	convertedHistogramMetricFamilies, err := reg.Gather()
	if err != nil {
		panic("unexpected behavior of custom test registry")
	}

	return convertedHistogramMetricFamilies
}

func findOldValueWithMetricFamily(oldPrometheusMetrics []*dto.MetricFamily, newM *dto.Metric, newMName string, newMType dto.MetricType) (float64, bool) {
	for _, oldMetric := range oldPrometheusMetrics {
		if newMName != *oldMetric.Name || newMType != *oldMetric.Type {
			continue
		}
		for _, oldM := range oldMetric.Metric {
			if checkEqualLabels(oldM.Label, newM.Label) {
				oldMetricValueFloat, succeed := getValueBasedOnType(*oldMetric.Type, *oldM)
				return oldMetricValueFloat, succeed
			}
		}
	}
	return 0.0, false
}

func getValueBasedOnType(metricType dto.MetricType, metric dto.Metric) (float64, bool) {
	switch metricType {
	case dto.MetricType_COUNTER:
		return *metric.Counter.Value, true
	case dto.MetricType_GAUGE:
		return *metric.Gauge.Value, true
	case dto.MetricType_HISTOGRAM:
		log.Errorf("This metric should already been converted to Gauge: metric.Histogram.String() = ", metric.Histogram.String())
		return 0.0, false
	case dto.MetricType_SUMMARY:
		log.Errorf("This metric should already been converted to Gauge: metric.Summary.String() = ", metric.Summary.String())
		return 0.0, false
	case dto.MetricType_UNTYPED:
		return *metric.Untyped.Value, true
	}
	return 0.0, false
}

func getLabels(metricLabels []*dto.LabelPair) ([]string, map[string]string) {
	labelKeysArray := []string{}
	labelMap := map[string]string{}
	for _, label := range metricLabels {
		labelKeysArray = append(labelKeysArray, *label.Name)
		labelMap[*label.Name] = *label.Value
	}
	return labelKeysArray, labelMap
}

func createNewMetricFamilies(newMetricName string, metricLabels []*dto.LabelPair, newMetricValue float64) *dto.MetricFamily {
	labelKeysArray, labelMap := getLabels(metricLabels)
	reg := prometheus.NewRegistry()
	metricFamily := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: newMetricName,
			Help: newMetricName,
		},
		labelKeysArray,
	)
	reg.MustRegister(metricFamily)
	metricFamily.With(labelMap).Set(newMetricValue)
	newMetricFamilies, err := reg.Gather()
	if err != nil || len(newMetricFamilies) != 1 {
		panic("unexpected behavior of custom test registry")
	}
	return newMetricFamilies[0]
}

func replaceHistogramSummaryToGauge(prometheusMetrics []*dto.MetricFamily) []*dto.MetricFamily {
	replacedMetricFamilies := []*dto.MetricFamily{}
	for _, pm := range prometheusMetrics {
		if *pm.Type == dto.MetricType_HISTOGRAM {
			newConvertHistogramToGaugeMetrics := convertHistogramToGauge(pm)
			for _, newGauge := range newConvertHistogramToGaugeMetrics {
				replacedMetricFamilies = append(replacedMetricFamilies, newGauge)
			}
		} else if *pm.Type == dto.MetricType_SUMMARY {
			newConvertSummaryToGaugeMetrics := convertSummaryToGauge(pm)
			for _, newGauge := range newConvertSummaryToGaugeMetrics {
				replacedMetricFamilies = append(replacedMetricFamilies, newGauge)
			}
		} else {
			replacedMetricFamilies = append(replacedMetricFamilies, pm)
		}
	}
	return replacedMetricFamilies
}

func convertSummaryToGauge(summaryMetricFamilies *dto.MetricFamily) []*dto.MetricFamily {
	reg := prometheus.NewRegistry()
	summaryQuantileMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: *summaryMetricFamilies.Name,
			Help: *summaryMetricFamilies.Help,
		},
		[]string{
			"quantile",
		},
	)
	summarySumMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: *summaryMetricFamilies.Name + "_sum",
			Help: *summaryMetricFamilies.Help,
		},
	)
	summaryCountMetric := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: *summaryMetricFamilies.Name + "_count",
			Help: *summaryMetricFamilies.Help,
		},
	)
	reg.MustRegister(summaryQuantileMetric)
	reg.MustRegister(summarySumMetric)
	reg.MustRegister(summaryCountMetric)
	for _, summaryMetric := range summaryMetricFamilies.Metric {
		summarySumValue := float64(*summaryMetric.Summary.SampleSum)
		summarySumMetric.Set(summarySumValue)
		summaryCountValue := float64(*summaryMetric.Summary.SampleCount)
		summaryCountMetric.Set(summaryCountValue)
		summaryQuantiles := summaryMetric.Summary.Quantile
		for _, hQuantile := range summaryQuantiles {
			summaryValue := float64(*hQuantile.Value)
			labelValue := strconv.FormatFloat(*hQuantile.Quantile, 'f', -1, 64)
			summaryQuantileMetric.WithLabelValues(labelValue).Set(summaryValue)
		}
	}

	convertedSummaryMetricFamilies, err := reg.Gather()
	if err != nil {
		panic("unexpected behavior of custom test registry")
	}

	return convertedSummaryMetricFamilies
}
