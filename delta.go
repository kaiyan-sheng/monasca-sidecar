// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateDelta(newPrometheusMetrics []*prometheusClient.MetricFamily, oldPrometheusMetrics []*prometheusClient.MetricFamily, rule SidecarRule) []*prometheusClient.MetricFamily {
	newDeltaMetrics := []*prometheusClient.MetricFamily{}
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if *pm.Name != rule.Parameters["name"] {
			continue
		}
		for _, newM := range pm.Metric {
			oldValueFloat, succeedOld := findOldValueWithMetricFamily(oldPrometheusMetrics, newM, *pm.Name, *pm.Type)
			if succeedOld {
				// calculate delta
				newValueFloat, succeedNew := getValueBasedOnType(*pm.Type, *newM)
				if !succeedNew {
					log.Warnf("Error getting values from new prometheus metric: %v", *pm.Name)
					continue
				}
				if *pm.Type == prometheusClient.MetricType_COUNTER && newValueFloat < oldValueFloat {
					log.Warnf("Counter %v has been reset", *pm.Name)
					continue
				}
				delta := newValueFloat - oldValueFloat

				// store delta metric into a new metric family
				newDeltaMetrics = append(newDeltaMetrics, createNewMetricFamilies(rule.Name, newM.Label, delta))
			}
		}
	}
	log.Infof("Successfully calculated delta for rule ", rule.Name)
	log.Debugf("Delta metrics = ", convertMetricFamiliesIntoTextString(newDeltaMetrics))
	return newDeltaMetrics
}
