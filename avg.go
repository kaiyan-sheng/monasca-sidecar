// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	dto "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateAvg(newPrometheusMetrics []*dto.MetricFamily, oldPrometheusMetrics []*dto.MetricFamily, rule SidecarRule) []*dto.MetricFamily {
	newAvgMetric := []*dto.MetricFamily{}
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
					// calculate avg
					newValueString, newValueFloat := getValueBasedOnType(newMType, *newM)
					if newValueString == "" {
						log.Errorf("Error getting values from new prometheus metric: %v", newMName)
						continue
					}
					avg := (newValueFloat + oldValueFloat) / 2.0
					// store avg metric into a new metric family
					newAvgMetric = append(newAvgMetric, createNewMetricFamilies(rule.Name, newM.Label, avg))
				}
			}
		}
	}
	return newAvgMetric
}
