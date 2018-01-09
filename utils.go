// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	log "github.hpe.com/kronos/kelog"
	"gopkg.in/yaml.v2"
	"sort"
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

func getPrometheusUrl(prometheusPort string, prometheusPath string) string {
	prefix := "http://localhost"
	if prometheusPath == "/" {
		prometheusUrl := prefix + ":" + prometheusPort
		return prometheusUrl
	}
	if strings.HasSuffix(prometheusPath, "/") {
		prometheusPath := prometheusPath[:(len(prometheusPath) - 1)]
		prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
		return prometheusUrl
	}
	prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
	return prometheusUrl
}

func convertDimensionsToHash(dimensions []Dimension) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", dimensions)))
	dimensionHash := h.Sum(nil)
	return dimensionHash
}

func sortDimensionsByKeys(dimensions map[string]string) map[string]string {
	sortedDimensions := map[string]string{}
	// get the list of keys and sort them
	keys := []string{}
	for key := range dimensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, val := range keys {
		sortedDimensions[val] = dimensions[val]
	}
	return sortedDimensions
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

func removeDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func dimensionsToString(dimensions []Dimension) string {
	if len(dimensions) == 0 {
		return ""
	}
	dimString := `{`
	for _, dim := range dimensions {
		dimKeyValue := dim.Key + "=" + dim.Value + ","
		dimString += dimKeyValue
	}
	dimString = dimString[0:len(dimString)-1] + "}"
	return dimString
}

func structNewMetricString(pm PrometheusMetric, newMetricValue float64, rule SidecarRule) string {
	newMetricName := rule.Name
	return "# HELP " + newMetricName + "\n" + "# TYPE gauge\n" + newMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(newMetricValue, 'e', 6, 64) + "\n"
}

func findDenominatorValue(prometheusMetrics []*dto.MetricFamily, numeratorLabels []*dto.LabelPair, denominatorName string) (string, float64) {
	for _, pm := range prometheusMetrics {
		if *pm.Name == denominatorName {
			for _, metric := range pm.Metric {
				if checkEqualLabels(numeratorLabels, metric.Label) {
					denominatorValueString, denominatorValueFloat := getValueBasedOnType(*pm.Type, *metric)
					return denominatorValueString, denominatorValueFloat
				}
			}
		}
	}
	return "", 0.0
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

func findOldValueWithMetricFamily(oldPrometheusMetrics []*dto.MetricFamily, newM *dto.Metric, newMName string, newMType dto.MetricType) (string, float64) {
	for _, oldMetric := range oldPrometheusMetrics {
		if newMName != *oldMetric.Name || newMType != *oldMetric.Type {
			continue
		}
		for _, oldM := range oldMetric.Metric {
			result := checkEqualLabels(oldM.Label, newM.Label)
			if result {
				oldMetricValueString, oldMetricValueFloat := getValueBasedOnType(*oldMetric.Type, *oldM)
				return oldMetricValueString, oldMetricValueFloat
			}
		}
	}
	return "", 0.0
}

func getValueBasedOnType(metricType dto.MetricType, metric dto.Metric) (string, float64) {
	switch metricType {
	case dto.MetricType_COUNTER:
		return metric.Counter.String(), *metric.Counter.Value
	case dto.MetricType_GAUGE:
		return metric.Gauge.String(), *metric.Gauge.Value
	case dto.MetricType_HISTOGRAM:
		log.Errorf("This metric should already been converted to Gague: metric.Histogram.String() = ", metric.Histogram.String())
		return "", 0.0
	case dto.MetricType_SUMMARY:
		return "", 0.0
	case dto.MetricType_UNTYPED:
		return metric.Untyped.String(), *metric.Untyped.Value
	}

	return "", 0.0
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

func createNewMetricFamilies(newMetricName string, metricLabels []*dto.LabelPair, newMetricValue float64) []*dto.MetricFamily {
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
	return newMetricFamilies
}

func replaceHistogramToGauge(prometheusMetrics []*dto.MetricFamily) []*dto.MetricFamily {
	replacedMetricFamilies := []*dto.MetricFamily{}
	for _, pm := range prometheusMetrics {
		if *pm.Type == dto.MetricType_HISTOGRAM {
			newConvertHistogramToGaugeMetrics := convertHistogramToGauge(pm)
			for _, newGauge := range newConvertHistogramToGaugeMetrics {
				replacedMetricFamilies = append(replacedMetricFamilies, newGauge)
			}
		} else {
			replacedMetricFamilies = append(replacedMetricFamilies, pm)
		}
	}
	return replacedMetricFamilies
}
