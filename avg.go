// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateAvg(newPrometheusMetrics PrometheusMetrics, oldPrometheusMetrics PrometheusMetrics, rule SidecarRule) (float64, error) {
	newAvgMetricString := ``
	// find old value and new value
	for _, pm := range newPrometheusMetrics {
		if pm.Name == rule.Parameters["name"] {
			oldValueString := findOldValue(oldPrometheusMetrics, pm)
			if oldValueString != "" {
				// calculate rate
				newValue, errNew := strconv.ParseFloat(pm.Value, 64)
				if errNew != nil {
					log.Errorf("Error converting strings to float64: %v", pm.Value)
					return 0.0, errNew
				}
				oldValue, errOld := strconv.ParseFloat(oldValueString, 64)
				if errOld != nil {
					log.Errorf("Error converting strings to float64: %v", oldValueString)
					return 0.0, errNew
				}
				avg := (newValue + oldValue) / 2.0
				// store rate metric into a new string
				newAvgMetricString += structNewStringRate(pm, avg)
			}
		}
	}
	return newAvgMetricString
}

func structNewStringAvg(pm PrometheusMetric, avg float64, avgRule SidecarRule) string {
	rateMetricName := avgRule.Name
	return "# HELP " + rateMetricName + "\n" + "# TYPE gauge \n" + rateMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(avg, 'e', 6, 64) + "\n"
}
