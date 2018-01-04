// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"crypto/sha256"
	"fmt"
	log "github.hpe.com/kronos/kelog"
	"gopkg.in/yaml.v2"
	"sort"
	"strings"
)

type SidecarRule struct {
	Name       string            `json:"name"`
	Function   string            `json:"function"`
	Parameters map[string]string `json:"parameters"`
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
	dimString := `{`
	for _, dim := range dimensions {
		dimString += dim.Key + "=" + dim.Value + ","
	}
	dimString += dimString[0:len(dimString)-1] + "}"
	return dimString
}
