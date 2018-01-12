// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateRate(newPrometheusMetrics []*prometheusClient.MetricFamily, oldPrometheusMetrics []*prometheusClient.MetricFamily, queryInterval float64, rule SidecarRule) []*prometheusClient.MetricFamily {
	newRateMetrics := []*prometheusClient.MetricFamily{}
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if *pm.Name == rule.Parameters["name"] {
			for _, newM := range pm.Metric {
				oldValueFloat, succeedOld := findOldValueWithMetricFamily(oldPrometheusMetrics, newM, *pm.Name, *pm.Type)
				if succeedOld {
					// calculate rate
					newValueFloat, succeedNew := getValueBasedOnType(*pm.Type, *newM)
					if !succeedNew {
						log.Errorf("Error getting values from new prometheus metric: %v", *pm.Name)
						continue
					}
					if *pm.Type == prometheusClient.MetricType_COUNTER && newValueFloat < oldValueFloat {
						log.Warnf("Counter %v has been reset", *pm.Name)
						continue
					}
					rate := (newValueFloat - oldValueFloat) / queryInterval

					// store rate metric into a new metric family
					newRateMetrics = append(newRateMetrics, createNewMetricFamilies(rule.Name, newM.Label, rate))
				}
			}
		}
	}
	return newRateMetrics
}
