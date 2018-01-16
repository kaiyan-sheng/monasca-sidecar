// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateAvg(newPrometheusMetrics []*prometheusClient.MetricFamily, oldPrometheusMetrics []*prometheusClient.MetricFamily, rule SidecarRule) []*prometheusClient.MetricFamily {
	newAvgMetrics := []*prometheusClient.MetricFamily{}
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if *pm.Name != rule.Parameters["name"] {
			continue
		}
		for _, newM := range pm.Metric {
			oldValueFloat, succeedOld := findOldValueWithMetricFamily(oldPrometheusMetrics, newM, *pm.Name, *pm.Type)
			if succeedOld {
				// calculate avg
				newValueFloat, succeedNew := getValueBasedOnType(*pm.Type, *newM)
				if !succeedNew {
					log.Warnf("Error getting values from new prometheus metric: %v", *pm.Name)
					continue
				}
				// check if MF is counter type, if it is check if it got reset
				if *pm.Type == prometheusClient.MetricType_COUNTER && newValueFloat < oldValueFloat {
					log.Warnf("Counter %v has been reset", *pm.Name)
					continue
				}
				avg := (newValueFloat + oldValueFloat) / 2.0
				// store avg metric into a new metric family
				newAvgMetrics = append(newAvgMetrics, createNewMetricFamilies(rule.Name, newM.Label, avg))
			}
		}
	}
	log.Infof("Successfully calculated avg for rule ", rule.Name)
	return newAvgMetrics
}
