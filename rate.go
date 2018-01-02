// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	log "github.hpe.com/kronos/kelog"
	"strconv"
)

func calculateRate(pm PrometheusMetric, oldValueString string, queryInterval float64) (float64, error) {
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
	rate := (newValue - oldValue) / queryInterval
	return rate, nil
}

func structNewStringRate(pm PrometheusMetric, rate float64) string {
	rateMetricName := pm.Name + "_per_second"
	return "# HELP " + rateMetricName + "\n" + "# TYPE gauge \n" + rateMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(rate, 'e', 6, 64) + "\n"
}

func dimensionsToString(dimensions []Dimension) string {
	dimString := `{`
	for _, dim := range dimensions {
		dimString += dim.Key + "=" + dim.Value + ","
	}
	dimString += dimString[0:len(dimString)-1] + "}"
	return dimString
}
