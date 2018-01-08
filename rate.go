// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateRate(newPrometheusMetrics []*dto.MetricFamily, oldPrometheusMetrics []*dto.MetricFamily, queryInterval float64, rule SidecarRule) []*dto.MetricFamily {
	newRateMetric := []*dto.MetricFamily{}
	newPrometheusMetricsWithNoHistogram := replaceHistogramToGauge(newPrometheusMetrics)
	oldPrometheusMetricsWithNoHistogram := replaceHistogramToGauge(oldPrometheusMetrics)
	// find old value and new value
	for _, pm := range newPrometheusMetricsWithNoHistogram {
		newMName := *pm.Name
		newMType := *pm.Type
		if *pm.Name == rule.Parameters["name"] {
			for _, newM := range pm.Metric {
				oldValueString, oldValueFloat := findOldValueWithMetricFamily(oldPrometheusMetricsWithNoHistogram, newM, newMName, newMType)
				if oldValueString != "" {
					// calculate rate
					newValueString, newValueFloat := getValueBasedOnType(newMType, *newM)
					if newValueString == "" {
						log.Errorf("Error getting values from new prometheus metric: %v", newMName)
						continue
					}
					rate := (newValueFloat - oldValueFloat) / queryInterval
					// store rate metric into a new metric family
					for _, newRate := range createNewRatePrometheus(rule.Name, newM.Label, rate) {
						newRateMetric = append(newRateMetric, newRate)
					}
				}
			}
		}
	}
	return newRateMetric
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

func createNewRatePrometheus(newMetricName string, metricLabels []*dto.LabelPair, newMetricValue float64) []*dto.MetricFamily {
	labelKeysArray, labelMap := getLabels(metricLabels)
	reg := prometheus.NewRegistry()
	rateMetric := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: newMetricName,
			Help: newMetricName,
		},
		labelKeysArray,
	)
	reg.MustRegister(rateMetric)
	rateMetric.With(labelMap).Set(newMetricValue)
	rateMetricFamilies, err := reg.Gather()
	if err != nil || len(rateMetricFamilies) != 1 {
		panic("unexpected behavior of custom test registry")
	}
	return rateMetricFamilies
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
