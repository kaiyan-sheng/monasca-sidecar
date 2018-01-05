// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateDeltaRatio(newPrometheusMetrics []PrometheusMetric, oldPrometheusMetrics []PrometheusMetric, rule SidecarRule) string {
	// deltaRatio = (newNumeratorValue - oldNumeratorValue) / (newDenominatorValue - oldDenominatorValue)
	newDeltaRatioMetricString := ``
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if pm.Name == rule.Parameters["numerator"] {
			oldValueString := findOldValue(oldPrometheusMetrics, pm)
			if oldValueString != "" {
				// calculate deltaNumeratorValue
				newNumeratorValue, errNew := strconv.ParseFloat(pm.Value, 64)
				if errNew != nil {
					log.Errorf("Error converting strings to float64: %v", pm.Value)
					continue
				}
				oldNumeratorValue, errOld := strconv.ParseFloat(oldValueString, 64)
				if errOld != nil {
					log.Errorf("Error converting strings to float64: %v", oldValueString)
					continue
				}
				deltaNumeratorValue := newNumeratorValue - oldNumeratorValue

				// get new denominator value
				newDenominatorValue, errDenomNew := findDenominatorValue(newPrometheusMetrics, pm.DimensionHash, rule.Parameters["denominator"])
				if errDenomNew != nil {
					continue
				}
				// get old denominator value
				oldDenominatorValue, errDenomOld := findDenominatorValue(oldPrometheusMetrics, pm.DimensionHash, rule.Parameters["denominator"])
				if errDenomOld != nil {
					continue
				}
				deltaDenominatorValue := newDenominatorValue - oldDenominatorValue

				// calculate ratio
				deltaRatioValue := deltaNumeratorValue / deltaDenominatorValue
				// store deltaRatio metric into a new string
				newDeltaRatioMetricString += structNewMetricString(pm, deltaRatioValue, rule)
			}
		}
	}
	return newDeltaRatioMetricString
}
