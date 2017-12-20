// (C) Copyright 2017 Hewlett Packard Enterprise Development LP

package main

import (
	"net/http"
	"fmt"
	"io/ioutil"
	"strings"
	"time"
	"strconv"
	"bytes"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	log "github.hpe.com/kronos/kelog"
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
	DimensionHash []byte `json:"hashcode"`
}

var oldRateMetricString = ``

func main() {
	//get namespace and pod name from environment variables
	podNamespace, ok := os.LookupEnv("SIDECAR_POD_NAMESPACE")
	if !ok {
		fmt.Printf("%s not set\n", "SIDECAR_POD_NAMESPACE")
		log.Errorf("%s not set\n", "SIDECAR_POD_NAMESPACE")
		os.Exit(1)
	}

	podName, ok := os.LookupEnv("SIDECAR_POD_NAME")
	if !ok {
		fmt.Printf("%s not set\n", "SIDECAR_POD_NAME")
		log.Errorf("%s not set\n", "SIDECAR_POD_NAMESPACE")
		os.Exit(1)
	}

	fmt.Printf("%s=%s\n", "SIDECAR_POD_NAME", podName)
	fmt.Printf("%s=%s\n", "SIDECAR_POD_NAMESPACE", podNamespace)

	//get annotations from pod kube config
	annotations := getPodAnnotations(podNamespace, podName)
	scrape := annotations["prometheus.io/scrape"]
	if scrape != "true" {
		fmt.Println("Scrape prometheus metrics is not enabled")
		fmt.Println("Please enable prometheus.io/scrape in annotations first")
		log.Errorf("Scrape prometheus metrics is not enabled. Please enable prometheus.io/scrape in annotations first.")
		os.Exit(1)
	}

	//get sidecar specific input parameters
	metricNames := annotations["sidecar/metric-names"]
	if metricNames == "" {
		log.Errorf("sidecar/metric-names can not be empty")
		os.Exit(1)
	}

	queryIntervalString := annotations["sidecar/query-interval"]
	if queryIntervalString == "" {
		log.Errorf("sidecar/query-interval can not be empty")
		os.Exit(1)
	}

	listenPort := annotations["sidecar/listen-port"]
	if queryIntervalString == "" {
		log.Errorf("sidecar/listenPort can not be empty")
		os.Exit(1)
	}

	metricNameArray := strings.Split(metricNames, ",")
	queryInterval, err := strconv.ParseFloat(queryIntervalString, 64)
	if err != nil {
		fmt.Println("Error converting \"sidecar/query-interval\" string to float64")
		log.Errorf("Error converting \"sidecar/query-interval\" string to float64")
	}
	if queryInterval <= 0.0 {
		log.Warnf("\"sidecar/query-interval\" can not be smaller or equal than zero. Set to default 30.0 seconds.")
		queryInterval = 30.0
	}

	//get prometheus url
	prometheusPort := annotations["prometheus.io/port"]
	if prometheusPort == "" {
		log.Errorf("\"prometheus.io/port\" can not be empty.")
		os.Exit(1)
	}

	prometheusPath := annotations["prometheus.io/path"]
	if prometheusPath == "" {
		prometheusPath = "/metrics"
		log.Infof("\"prometheus.io/path\" is empty, set to default \"/metrics\" for prometheus path.")
	}

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
			oldValueString := findOldValue(oldPrometheusMetrics, pm)
			if oldValueString != "" {
				rate, errRate := calculateRate(pm, oldValueString, queryInterval)
				if errRate != nil {
					log.Errorf("Failed to calculate rate for metric %v", pm.Name)
					continue
				}
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
		log.Infof("Prometheus metrics does not include %v", metricName)
		return prometheusMetrics
	}

	splitWithName := strings.Split(respBody, "# HELP " + metricName)
	metricString := strings.Split(splitWithName[1], "# HELP")[0]

	// Convert a string into structure
	metricStringLines := strings.Split(metricString, "\n")
	// Conver each line
	for _, i := range(metricStringLines[2:]) {
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
				splitDims := strings.Split(dimensions, ",")
				for _, d := range(splitDims) {
					split_each_dim := strings.Split(d, "=")
					dim := Dimension{Key: split_each_dim[0], Value: split_each_dim[1]}
					metricDimensions = append(metricDimensions, dim)
				}
				pm := PrometheusMetric{Name: iMetricName, Value: metricValue, Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
				prometheusMetrics = append(prometheusMetrics, pm)
			} else {
				iMetricName := metricSplit[0]
				pm := PrometheusMetric{Name: iMetricName, Value: metricValue, Dimensions: metricDimensions, DimensionHash: convertDimensionsToHash(metricDimensions)}
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
		log.Errorf("Error scraping prometheus endpoint")
		os.Exit(1)
	}
	if resp.ContentLength == 0 {
		fmt.Println("No prometheus metric from ", prometheusUrl)
		log.Warnf("No prometheus metric from %v", prometheusUrl)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	return string(respBody)
}

func findOldValue(oldPrometheusMetrics []PrometheusMetric, newPrometheusMetric PrometheusMetric) string {
	for _, oldMetric := range(oldPrometheusMetrics) {
		if newPrometheusMetric.Name != oldMetric.Name {
			continue
		}
		if bytes.Equal(newPrometheusMetric.DimensionHash, oldMetric.DimensionHash) {
			return oldMetric.Value
		}
	}
	fmt.Println("Can not find previous value for metric ", newPrometheusMetric.Name)
	log.Warnf("Can not find previous value for metric ", newPrometheusMetric.Name)
	return ""
}

func pushPrometheusMetricsString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, oldRateMetricString) // send data to client side
}

func getPodAnnotations(namespace string, podName string) map[string]string {
	annotations := map[string]string{}
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
		fmt.Println("Pod not found")
		log.Errorf("Pod %v not found in namespace %v.", podName, namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		fmt.Println("Error getting pod %v", statusError.ErrStatus.Message)
		log.Errorf("Error getting pod %v in namespace %v: %v", podName, namespace, statusError.ErrStatus.Message)
	} else if err != nil {
		panic(err.Error())
	} else {
		fmt.Printf("Found pod\n")
		log.Infof("Found pod %v in namespace %v", podName, namespace)
		annotations = podGet.Annotations
	}
	return annotations
}
