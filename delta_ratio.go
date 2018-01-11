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
			for _, newM := range pm.Metric {
				oldNumeratorValueFloat, succeedOldNumerator := findOldValueWithMetricFamily(oldPrometheusMetrics, newM, *pm.Name, *pm.Type)
				if succeedOldNumerator {
					// calculate deltaNumeratorValue
					newNumeratorValueFloat, succeedNewNumerator := getValueBasedOnType(*pm.Type, *newM)
					if !succeedNewNumerator {
						log.Errorf("Error getting new numerator value from new prometheus metric: %v", *pm.Name)
						continue
					}
					deltaNumeratorValue := newNumeratorValueFloat - oldNumeratorValueFloat
					if *pm.Type == dto.MetricType_COUNTER {
						if deltaNumeratorValue < 0 {
							log.Warnf("Counter %v has been reset", rule.Parameters["numerator"])
							continue
						}
					}

					// get new denominator value
					newDenominatorValueFloat, succeedNewDenominator := findDenominatorValue(newPrometheusMetrics, newM.Label, rule.Parameters["denominator"])
					if !succeedNewDenominator {
						log.Errorf("Error getting new denominator value from new prometheus metric: %v", *pm.Name)
						continue
					}
					// get old denominator value
					oldDenominatorValueFloat, succeedOldDenominator := findDenominatorValue(oldPrometheusMetrics, newM.Label, rule.Parameters["denominator"])
					if !succeedOldDenominator {
						log.Errorf("Error getting old denominator value from old prometheus metric: %v", *pm.Name)
						continue
					}
					deltaDenominatorValue := newDenominatorValueFloat - oldDenominatorValueFloat
					if *pm.Type == dto.MetricType_COUNTER {
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
