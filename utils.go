// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	log "github.hpe.com/kronos/kelog"
	"gopkg.in/yaml.v2"
	"sort"
	"strconv"
	"strings"
)

type SidecarRule struct {
	Name       string            `yaml:"metricName"`
	Function   string            `yaml:"function"`
	Parameters map[string]string `yaml:"parameters"`
}

func stringBetween(value string, a string, b string) string {
	// Get substring between two strings.
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		log.Warnf("Start chars do not exist in original string")
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		log.Warnf("End chars do not exist in original string")
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	if posFirstAdjusted >= posLast {
		log.Warnf("Start chars is on the right side of end chars")
		return ""
	}
	return value[posFirstAdjusted:posLast]
}

func getPrometheusUrl(prometheusPort string, prometheusPath string) string {
	prefix := "http://localhost"
	if prometheusPath == "/" {
		prometheusUrl := prefix + ":" + prometheusPort
		return prometheusUrl
	}
	if strings.HasSuffix(prometheusPath, "/") {
		prometheusPath := prometheusPath[:(len(prometheusPath) - 1)]
		prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
		return prometheusUrl
	}
	prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
	return prometheusUrl
}

func convertDimensionsToHash(dimensions []Dimension) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", dimensions)))
	dimensionHash := h.Sum(nil)
	return dimensionHash
}

func sortDimensionsByKeys(dimensions map[string]string) map[string]string {
	sortedDimensions := map[string]string{}
	// get the list of keys and sort them
	keys := []string{}
	for key := range dimensions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, val := range keys {
		sortedDimensions[val] = dimensions[val]
	}
	return sortedDimensions
}

func parseYamlSidecarRules(rules string) []SidecarRule {
	var ruleStruct []SidecarRule
	source := []byte(rules)
	err := yaml.Unmarshal(source, &ruleStruct)
	if err != nil {
		log.Fatalf("Error parsing sidecar rules: ", err)
	}
	return ruleStruct
}

func removeDuplicates(elements []string) []string {
	// Use map to record duplicates as we find them.
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if encountered[elements[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[elements[v]] = true
			// Append to result slice.
			result = append(result, elements[v])
		}
	}
	// Return the new slice.
	return result
}

func dimensionsToString(dimensions []Dimension) string {
	if len(dimensions) == 0 {
		return ""
	}
	dimString := `{`
	for _, dim := range dimensions {
		dimKeyValue := dim.Key + "=" + dim.Value + ","
		dimString += dimKeyValue
	}
	dimString = dimString[0:len(dimString)-1] + "}"
	return dimString
}

func structNewMetricString(pm PrometheusMetric, newMetricValue float64, rule SidecarRule) string {
	newMetricName := rule.Name
	return "# HELP " + newMetricName + "\n" + "# TYPE gauge\n" + newMetricName + dimensionsToString(pm.Dimensions) + " " + strconv.FormatFloat(newMetricValue, 'e', 6, 64) + "\n"
}

func findDenominatorValue(prometheusMetrics []PrometheusMetric, dimensionHash []byte, denominatorName string) (float64, error) {
	for _, pm := range prometheusMetrics {
		if pm.Name == denominatorName && bytes.Equal(dimensionHash, pm.DimensionHash) {
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
