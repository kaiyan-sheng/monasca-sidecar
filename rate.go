// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

package main

import (
	"strconv"
	"fmt"
)

func calculateRate(pm PrometheusMetric, oldValueString string, queryInterval float64) float64 {
	newValue, errNew := strconv.ParseFloat(pm.Value, 64)
	if errNew != nil {
		fmt.Println("Error converting strings to float64")
	}
	oldValue, errOld := strconv.ParseFloat(oldValueString, 64)
	if errOld != nil {
		fmt.Println("Error converting strings to float64")
	}
	rate := (newValue - oldValue) / queryInterval
	return rate
}

func structNewStringRate(pm PrometheusMetric, rate float64) string {
	rateMetricName := pm.Name + "_per_second"
	return `# HELP request_count_per_sec Counts per second requests by method and path
# TYPE request_count_per_sec gauge` + "\n" + rateMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(rate, 'e', 6, 64) + "\n"
}

