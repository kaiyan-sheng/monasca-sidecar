// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	dto "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateDeltaRatio(newPrometheusMetrics []*dto.MetricFamily, oldPrometheusMetrics []*dto.MetricFamily, rule SidecarRule) []*dto.MetricFamily {
	// deltaRatio = (newNumeratorValue - oldNumeratorValue) / (newDenominatorValue - oldDenominatorValue)
	newDeltaRatioMetric := []*dto.MetricFamily{}
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if *pm.Name == rule.Parameters["numerator"] {
			newMName := *pm.Name
			newMType := *pm.Type
			for _, newM := range pm.Metric {
				oldNumeratorValueString, oldNumeratorValueFloat := findOldValueWithMetricFamily(oldPrometheusMetrics, newM, newMName, newMType)
				if oldNumeratorValueString != "" {
					// calculate deltaNumeratorValue
					newNumeratorValueString, newNumeratorValueFloat := getValueBasedOnType(newMType, *newM)
					if newNumeratorValueString == "" {
						log.Errorf("Error getting new numerator value from new prometheus metric: %v", newMName)
						continue
					}
					deltaNumeratorValue := newNumeratorValueFloat - oldNumeratorValueFloat
					if newMType == dto.MetricType_COUNTER {
						if deltaNumeratorValue < 0 {
							log.Warnf("Counter %v has been reset", rule.Parameters["numerator"])
							continue
						}
					}

					// get new denominator value
					newDenominatorValueString, newDenominatorValueFloat := findDenominatorValue(newPrometheusMetrics, newM.Label, rule.Parameters["denominator"])
					if newDenominatorValueString == "" {
						log.Errorf("Error getting new denominator value from new prometheus metric: %v", newMName)
						continue
					}
					// get old denominator value
					oldDenominatorValueString, oldDenominatorValueFloat := findDenominatorValue(oldPrometheusMetrics, newM.Label, rule.Parameters["denominator"])
					if oldDenominatorValueString == "" {
						log.Errorf("Error getting old denominator value from old prometheus metric: %v", newMName)
						continue
					}
					deltaDenominatorValue := newDenominatorValueFloat - oldDenominatorValueFloat
					if newMType == dto.MetricType_COUNTER {
						if deltaDenominatorValue < 0 {
							log.Warnf("Counter %v has been reset", rule.Parameters["denominator"])
							continue
						}
					}
					// calculate ratio
					deltaRatioValue := deltaNumeratorValue / deltaDenominatorValue
					// store delta ratio metric into a new metric family
					newDeltaRatioMetric = append(newDeltaRatioMetric, createNewMetricFamilies(rule.Name, newM.Label, deltaRatioValue))
				}
			}
		}
	}
	return newDeltaRatioMetric
}
