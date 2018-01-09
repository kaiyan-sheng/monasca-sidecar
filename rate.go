// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
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
					newRateMetric = append(newRateMetric, createNewMetricFamilies(rule.Name, newM.Label, rate))
				}
			}
		}
	}
	return newRateMetric
}
