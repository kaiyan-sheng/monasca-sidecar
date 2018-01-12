// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateRatio(prometheusMetrics []*prometheusClient.MetricFamily, rule SidecarRule) []*prometheusClient.MetricFamily {
	newRatioMetrics := []*prometheusClient.MetricFamily{}
	for _, pm := range prometheusMetrics {
		if *pm.Name == rule.Parameters["numerator"] {
			// get denominator value
			for _, metric := range pm.Metric {
				numeratorValueFloat, succeedNumerator := getValueBasedOnType(*pm.Type, *metric)
				if !succeedNumerator {
					log.Errorf("Error getting numerator value from prometheus metric: %v", *pm.Name)
					continue
				}
				denominatorValueFloat, succeedDenominator := findDenominatorValue(prometheusMetrics, metric.Label, rule.Parameters["denominator"])
				if !succeedDenominator {
					log.Errorf("Error getting denominator value from prometheus metric: %v", *pm.Name)
					continue
				}
				if denominatorValueFloat == 0.0 {
					log.Warnf("Denominator value from metric %v with labels %v cannot be zero", *pm.Name, metric.Label)
					continue
				}
				ratio := numeratorValueFloat / denominatorValueFloat
				// store ratio metric into a new metric family
				newRatioMetrics = append(newRatioMetrics, createNewMetricFamilies(rule.Name, metric.Label, ratio))
			}
		}
	}
	log.Infof("Successfully calculated ratio for rule ", rule.Name)
	return newRatioMetrics
}
