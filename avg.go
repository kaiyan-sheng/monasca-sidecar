// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateAvg(newPrometheusMetrics []PrometheusMetric, oldPrometheusMetrics []PrometheusMetric, rule SidecarRule) string {
	newAvgMetricString := ``
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if pm.Name == rule.Parameters["name"] {
			oldValueString := findOldValue(oldPrometheusMetrics, pm)
			if oldValueString != "" {
				// calculate avg
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
				avg := (newValue + oldValue) / 2.0
				// store avg metric into a new string
				newAvgMetricString += structNewStringAvg(pm, avg, rule)
			}
		}
	}
	return newAvgMetricString
}

func structNewStringAvg(pm PrometheusMetric, avg float64, avgRule SidecarRule) string {
	avgMetricName := avgRule.Name
	return "# HELP " + avgMetricName + "\n" + "# TYPE gauge \n" + avgMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(avg, 'e', 6, 64) + "\n"
}
