// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

package main

import (
	"strings"
	"fmt"
	"crypto/sha256"
)

func stringBetween(value string, a string, b string) string {
	// Get substring between two strings.
	posFirst := strings.Index(value, a)
	if posFirst == -1 {
		fmt.Println("Start chars do not exist in original string")
		return ""
	}
	posLast := strings.Index(value, b)
	if posLast == -1 {
		fmt.Println("End chars do not exist in original string")
		return ""
	}
	posFirstAdjusted := posFirst + len(a)
	if posFirstAdjusted >= posLast {
		fmt.Println("Start chars is on the right side of end chars")
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
		prometheusUrl := prefix + prometheusPath + ":" + prometheusPort
		return prometheusUrl
	}
	prometheusUrl := prefix + prometheusPath + ":" + prometheusPort
	return prometheusUrl
}

func convertDimensionsToHash(dimensions map[string]string) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", dimensions)))
	dimensionHash := h.Sum(nil)
	return dimensionHash
}
