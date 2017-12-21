// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

package main

import (
	"strings"
	log "github.hpe.com/kronos/kelog"
	"crypto/sha256"
	"sort"
	"fmt"
)

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

func getPrometheusUrl (prometheusPort string, prometheusPath string) string {
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
	prometheusUrl := prefix  + ":" + prometheusPort + prometheusPath
	return prometheusUrl
}

func convertDimensionsToHash(dimensions map[string]string) []byte {
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
