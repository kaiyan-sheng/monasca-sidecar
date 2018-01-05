// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateRate(newPrometheusMetrics []PrometheusMetric, oldPrometheusMetrics []PrometheusMetric, queryInterval float64, rule SidecarRule) string {
	newRateMetricString := ``
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if pm.Name == rule.Parameters["name"] {
			oldValueString := findOldValue(oldPrometheusMetrics, pm)
			if oldValueString != "" {
				// calculate rate
				newValue, errNew := strconv.ParseFloat(pm.Value, 64)
				if errNew != nil {
					log.Errorf("Error converting strings to float64: %v", pm.Value)
					continue
				}
				oldValue, errOld := strconv.ParseFloat(oldValueString, 64)
				if errOld != nil {
					log.Errorf("Error converting strings to float64: %v", oldValueString)
					continue
				}
				rate := (newValue - oldValue) / queryInterval
				// store rate metric into a new string
				newRateMetricString += structNewMetricString(pm, rate, rule)
			}
		}
	}
	return newRateMetricString
}

func findOldValue(oldPrometheusMetrics []PrometheusMetric, newPrometheusMetric PrometheusMetric) string {
	for _, oldMetric := range oldPrometheusMetrics {
		if newPrometheusMetric.Name != oldMetric.Name {
			continue
		}
		if bytes.Equal(newPrometheusMetric.DimensionHash, oldMetric.DimensionHash) {
			return oldMetric.Value
		}
	}
	return ""
}
