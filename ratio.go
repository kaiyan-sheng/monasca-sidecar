// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateRatio(prometheusMetrics []PrometheusMetric, rule SidecarRule) string {
	newRatioMetricString := ``
	for _, pm := range prometheusMetrics {
		if pm.Name == rule.Parameters["numerator"] {
			// get denominator value
			numeratorValue, errNumerator := strconv.ParseFloat(pm.Value, 64)
			if errNumerator != nil {
				log.Errorf("Error converting strings to float64: %v", pm.Value)
				continue
			}
			denominatorValue, err := findDenominatorValue(prometheusMetrics, pm.DimensionHash, rule.Parameters["denominator"])
			if err != nil {
				continue
			}
			ratio := numeratorValue / denominatorValue
			// store ratio metric into a new string
			newRatioMetricString += structNewMetricString(pm, ratio, rule)
		}
	}
	return newRatioMetricString
}
