// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
	"strconv"
	"crypto/sha256"
	"bytes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
)

type Dimension struct {
	Key string `json:"key"`
	Value string `json:"value"`
}

type DimensionList []Dimension

type PrometheusMetric struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	Dimensions DimensionList `json:"dimensions"`
}

var oldRateMetricString = ``

func main() {
	//get namespace and pod name from environment variables
	podNamespace, ok := os.LookupEnv("SIDECAR_POD_NAMESPACE")
	if !ok {
		fmt.Printf("%s not set\n", "SIDECAR_POD_NAMESPACE")
	} else {
		fmt.Printf("%s=%s\n", "SIDECAR_POD_NAMESPACE", podNamespace)
	}

	podName, ok := os.LookupEnv("SIDECAR_POD_NAME")
	if !ok {
		fmt.Printf("%s not set\n", "SIDECAR_POD_NAME")
	} else {
		fmt.Printf("%s=%s\n", "SIDECAR_POD_NAME", podName)
	}

	annotations := getPodAnnotations(podNamespace, podName)
	scrape := annotations["prometheus.io/scrape"]
	if scrape != "true" {
		fmt.Println("Scrape prometheus metrics is not enabled")
		fmt.Println("Please enable prometheus.io/scrape in annotations first")
		os.Exit(1)
	}
	metricNames := annotations["sidecar/metric-names"]
	queryIntervalString := annotations["sidecar/query-interval"]
	listenPort := annotations["sidecar/listen-port"]

	metricNameArray := strings.Split(metricNames, ",")
	queryInterval, err := strconv.ParseFloat(queryIntervalString, 64)
	if err != nil {
		fmt.Println("Error converting strings to float64")
	}
	//get prometheus url
	prometheusPort := annotations["prometheus.io/port"]
	prometheusPath := annotations["prometheus.io/path"]
	prometheusUrl := getPrometheusUrl (prometheusPort, prometheusPath)

	// get prometheus metric response body
	respBody := getPrometheusMetrics(prometheusUrl)
	oldRateMetricString = respBody

	// extract information about the metric into structure
	oldPrometheusMetrics := []PrometheusMetric{}
	for _, metricName := range(metricNameArray) {
		oldPrometheusMetrics = responseBodyToStructure(respBody, metricName, oldPrometheusMetrics)
	}

	// start web server
	http.HandleFunc("/", pushPrometheusMetricsString) // set router
	go http.ListenAndServe(":" + listenPort, nil) // set listen port

	// Infinite for loop to scrape prometheus metrics and calculate rate every 30 seconds
	for {
		newRateMetricString := ``
		// sleep for 30 seconds
		fmt.Println("Starting sleeping for 30 seconds")
		time.Sleep(time.Second * time.Duration(queryInterval))
		fmt.Println("Done sleeping")
		fmt.Println("----------")
		// get a new set of prometheus metrics
		newRespBody := getPrometheusMetrics(prometheusUrl)
		// extract information about the metric into structure
		newPrometheusMetrics := []PrometheusMetric{}
		for _, metricName := range(metricNameArray) {
			newPrometheusMetrics = responseBodyToStructure(newRespBody, metricName, newPrometheusMetrics)
		}

		// compare dimensions and calculate rate
		for _, pm := range(newPrometheusMetrics) {
			oldValueString := findOldValue(oldPrometheusMetrics, pm.Dimensions, pm.Name)
			if oldValueString != "" {
				rate := calculateRate(pm, oldValueString, queryInterval)
				fmt.Println("rate = ", rate)
				// store rate metric into a new string
				newRateMetricString += structNewStringRate(pm, rate)
			}
		}
		fmt.Println("----------")

		// set current to old to prepare new collection in next for loop
		oldPrometheusMetrics = newPrometheusMetrics
		oldRateMetricString = newRespBody + newRateMetricString
	}
}

func responseBodyToStructure(respBody string, metricName string, prometheusMetrics []PrometheusMetric) []PrometheusMetric {
	// Find metric name and parse the response body string
	fmt.Println("metricName = ", metricName)
	if !strings.Contains(respBody, metricName) {
		fmt.Println("Prometheus metrics does not include ", metricName)
		return prometheusMetrics
	}
	split_with_name := strings.Split(respBody, "# HELP " + metricName)
	metricString := strings.Split(split_with_name[1], "# HELP")[0]
	// Convert a string into structure
	metricStringLines := strings.Split(metricString, "\n")

	for _, i := range(metricStringLines[2:]) {
		fmt.Println(i)
		metricSplit := strings.Split(i, " ")
		if len(metricSplit) > 1  {
			metricDimensions := []Dimension{}
			//get metric value
			metricValue := metricSplit[1]
			//get metric name
			if strings.ContainsAny(string(i), "{") {
				iMetricName := strings.Split(string(i), "{")[0]
				// get dimensions
				dimensions := stringBetween(string(i), "{", "}")
				split_dims := strings.Split(dimensions, ",")
				for _, d := range(split_dims) {
					split_each_dim := strings.Split(d, "=")
					dim := Dimension{Key: split_each_dim[0], Value: split_each_dim[1]}
					metricDimensions = append(metricDimensions, dim)
				}
				pm := PrometheusMetric{Name: iMetricName, Value: metricValue, Dimensions: metricDimensions}
				prometheusMetrics = append(prometheusMetrics, pm)
			} else {
				iMetricName := metricSplit[0]
				pm := PrometheusMetric{Name: iMetricName, Value: metricValue, Dimensions: metricDimensions}
				prometheusMetrics = append(prometheusMetrics, pm)
			}

		}
	}
	return prometheusMetrics
}

func getPrometheusMetrics(prometheusUrl string) string {
	resp, err := http.Get(prometheusUrl)
	if err != nil {
		fmt.Println("Error scraping prometheus endpoint")
	}
	if resp.ContentLength == 0 {
		fmt.Println("No prometheus metric from ", prometheusUrl)
	}
	fmt.Println("status code = ", resp.StatusCode)
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	return string(respBody)
}

func findOldValue(oldPrometheusMetrics []PrometheusMetric, newDimensions []Dimension, metricName string) string {
	hNew := sha256.New()
	hNew.Write([]byte(fmt.Sprintf("%v", newDimensions)))
	newDimensionHash :=  hNew.Sum(nil)
	for _, oldMetric := range(oldPrometheusMetrics) {
		if metricName != oldMetric.Name {
			continue
		}
		hOld := sha256.New()
		hOld.Write([]byte(fmt.Sprintf("%v", oldMetric.Dimensions)))
		oldDimensionHash :=  hOld.Sum(nil)
		if bytes.Equal(newDimensionHash, oldDimensionHash) {
			return oldMetric.Value
		}
	}
	return ""
}

func dimensionsToString(dimensions []Dimension) string {
	dimString := `{`
	for _, dim := range (dimensions) {
		dimString += dim.Key + "=" + dim.Value + ","
	}
	dimString += dimString[0:len(dimString)-1] + "}"
	return dimString
}

func pushPrometheusMetricsString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, oldRateMetricString) // send data to client side
}

func getPodAnnotations(namespace string, podName string) map[string]string {
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}
	// creates the clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	podGet, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		fmt.Printf("Pod not found\n")
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Printf("Error getting pod %v\n", statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod\n")
		annotations := podGet.Annotations
		return annotations
	}
	return map[string]string{}
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
