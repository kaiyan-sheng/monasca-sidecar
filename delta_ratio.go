// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
)

func calculateDeltaRatio(newPrometheusMetrics []*prometheusClient.MetricFamily, oldPrometheusMetrics []*prometheusClient.MetricFamily, rule SidecarRule) []*prometheusClient.MetricFamily {
	// deltaRatio = (newNumeratorValue - oldNumeratorValue) / (newDenominatorValue - oldDenominatorValue)
	newDeltaRatioMetrics := []*prometheusClient.MetricFamily{}
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
					if *pm.Type == prometheusClient.MetricType_COUNTER && newNumeratorValueFloat < oldNumeratorValueFloat {
						log.Warnf("Counter %v has been reset", rule.Parameters["numerator"])
						continue
					}
					deltaNumeratorValue := newNumeratorValueFloat - oldNumeratorValueFloat

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
					if *pm.Type == prometheusClient.MetricType_COUNTER && newDenominatorValueFloat < oldDenominatorValueFloat {
						log.Warnf("Counter %v has been reset", rule.Parameters["denominator"])
						continue
					}
					deltaDenominatorValue := newDenominatorValueFloat - oldDenominatorValueFloat
					if deltaDenominatorValue == 0.0 {
						log.Infof("Delta value of denominator from metric %v with labels %v cannot be zero", *pm.Name, newM.Label)
						continue
					}

					// calculate ratio
					deltaRatioValue := deltaNumeratorValue / deltaDenominatorValue
					// store delta ratio metric into a new metric family
					newDeltaRatioMetrics = append(newDeltaRatioMetrics, createNewMetricFamilies(rule.Name, newM.Label, deltaRatioValue))
				}
			}
		}
	}
	return newDeltaRatioMetrics
}
