// (C) Copyright 2018 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	"errors"
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateRatio(prometheusMetrics []PrometheusMetric, rule SidecarRule) string {
	newRatioMetricString := ``
	for _, pm := range prometheusMetrics {
		if pm.Name == rule.Parameters["numerator"] {
			// get numerator value
			numeratorValue, errNumerator := strconv.ParseFloat(pm.Value, 64)
			if errNumerator != nil {
				log.Errorf("Error converting strings to float64: %v", pm.Value)
				continue
			}
			denominatorValue, err := findDenominatorValue(prometheusMetrics, pm.DimensionHash, rule)
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

func findDenominatorValue(prometheusMetrics []PrometheusMetric, dimensionHash []byte, rule SidecarRule) (float64, error) {
	for _, pm := range prometheusMetrics {
		if pm.Name == rule.Parameters["denominator"] && bytes.Equal(dimensionHash, pm.DimensionHash) {
			// get denominator value
			denominatorValue, errDenominator := strconv.ParseFloat(pm.Value, 64)
			if errDenominator != nil {
				log.Errorf("Error converting strings to float64: %v", pm.Value)
			}
			return denominatorValue, errDenominator
		}
	}
	return 0.0, errors.New("Cannot find the denominator metric with the same dimensions as numerator")
}
