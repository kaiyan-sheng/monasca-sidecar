// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	dto "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateRatio(prometheusMetrics []*dto.MetricFamily, rule SidecarRule) []*dto.MetricFamily {
	newRatioMetric := []*dto.MetricFamily{}
	for _, pm := range prometheusMetrics {
		if *pm.Name == rule.Parameters["numerator"] {
			// get denominator value
			for _, metric := range pm.Metric {
				numeratorValueString, numeratorValueFloat := getValueBasedOnType(*pm.Type, *metric)
				if numeratorValueString == "" {
					log.Errorf("Error getting numerator value from prometheus metric: %v", *pm.Name)
					continue
				}
				denominatorValueString, denominatorValueFloat := findDenominatorValue(prometheusMetrics, metric.Label, rule.Parameters["denominator"])
				if denominatorValueString == "" && denominatorValueFloat == 0.0 {
					log.Errorf("Error getting denominator value from prometheus metric: %v", *pm.Name)
					continue
				}
				ratio := numeratorValueFloat / denominatorValueFloat
				// store ratio metric into a new metric family
				newRatioMetric = append(newRatioMetric, createNewMetricFamilies(rule.Name, metric.Label, ratio))
			}
		}
	}
	log.Infof("Successfully calculated ratio for rule ", rule.Name)
	return newRatioMetric
}
