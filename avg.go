// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateAvg(pm PrometheusMetric, oldValueString string) (float64, error) {
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
	return avg, nil
}

func structNewStringAvg(pm PrometheusMetric, avg float64, avgRule SidecarRule) string {
	rateMetricName := avgRule.Name
	return "# HELP " + rateMetricName + "\n" + "# TYPE gauge \n" + rateMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(avg, 'e', 6, 64) + "\n"
}
