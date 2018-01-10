// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"
	dto "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var oldPrometheusMetricString = ``

func main() {
	val, ok := os.LookupEnv("LOG_LEVEL")
	logLevelEnv := "info"
	if ok {
		logLevelEnv = val
	}
	logLevel := strings.ToLower(logLevelEnv)
	if logLevel != "" {
		log.Printf("Setting global log level to '%s'", logLevel)
		log.SetLevelString(logLevel)
	}

	//get namespace and pod name from environment variables
	podNamespace, ok := os.LookupEnv("SIDECAR_POD_NAMESPACE")
	if !ok {
		log.Errorf("%s not set\n", "SIDECAR_POD_NAMESPACE")
		os.Exit(1)
	}

	podName, ok := os.LookupEnv("SIDECAR_POD_NAME")
	if !ok {
		log.Errorf("%s not set\n", "SIDECAR_POD_NAME")
		os.Exit(1)
	}
	log.Infof("%s=%s\n", "SIDECAR_POD_NAME", podName)
	log.Infof("%s=%s\n", "SIDECAR_POD_NAMESPACE", podNamespace)

	//get annotations from pod kube config
	annotations, errGetAnnotations := getPodAnnotations(podNamespace, podName)
	if errGetAnnotations != nil {
		os.Exit(1)
	}
	scrape := annotations["prometheus.io/scrape"]
	if scrape != "true" {
		log.Fatalf("Scrape prometheus metrics is not enabled. Please enable prometheus.io/scrape in annotations first.")
	}

	//get sidecar specific input parameters
	queryIntervalString := annotations["sidecar/query-interval"]
	if queryIntervalString == "" {
		log.Fatalf("sidecar/query-interval can not be empty")
	}

	listenPort := annotations["sidecar/listen-port"]
	if queryIntervalString == "" {
		log.Fatalf("sidecar/listenPort can not be empty")
	}

	rules := annotations["sidecar/rules"]
	if rules == "" {
		log.Fatalf("sidecar/rules can not be empty")
	}
	log.Infof("rules = %s\n", rules)

	sidecarRules := parseYamlSidecarRules(rules)

	queryInterval, err := strconv.ParseFloat(queryIntervalString, 64)
	if queryInterval <= 0.0 || err != nil {
		log.Warnf("Error converting \"sidecar/query-interval\". Set queryInterval to default 30.0 seconds.")
		queryInterval = 30.0
	}

	// get prometheus url and prometheus metric response body
	oldPrometheusMetrics := getPrometheusMetrics(annotations)
	oldPrometheusMetricString = convertMetricFamiliesIntoTextString(oldPrometheusMetrics)

	// start web server
	http.HandleFunc("/", pushPrometheusMetricsString) // set router
	go http.ListenAndServe(":"+listenPort, nil)       // set listen port

	// Infinite for loop to scrape prometheus metrics and calculate rate every 30 seconds
	for {
		newRateMetricStringTotal := ``
		newAvgMetricStringTotal := ``
		newRatioMetricStringTotal := ``
		newDeltaRatioMetricStringTotal := ``

		// sleep for 30 seconds or how long queryInterval is
		time.Sleep(time.Second * time.Duration(queryInterval))

		// get a new set of prometheus metrics
		newPrometheusMetrics := getPrometheusMetrics(annotations)

		newPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(newPrometheusMetrics)
		oldPrometheusMetricsWithNoHistogramSummary := replaceHistogramSummaryToGauge(oldPrometheusMetrics)
		// calculate by each sidecar rule
		for _, rule := range sidecarRules {
			switch rule.Function {
			case "rate":
				newRateMetrics := calculateRate(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, queryInterval, rule)
				newRateMetricString := convertMetricFamiliesIntoTextString(newRateMetrics)
				newRateMetricStringTotal += newRateMetricString
			case "avg":
				newAvgMetrics := calculateAvg(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, rule)
				newAvgMetricString := convertMetricFamiliesIntoTextString(newAvgMetrics)
				newAvgMetricStringTotal += newAvgMetricString
			case "ratio":
				newRatioMetrics := calculateRatio(newPrometheusMetricsWithNoHistogramSummary, rule)
				newRatioMetricString := convertMetricFamiliesIntoTextString(newRatioMetrics)
				newRatioMetricStringTotal += newRatioMetricString
			case "deltaRatio":
				newDeltaRatioMetrics := calculateDeltaRatio(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, rule)
				newDeltaRatioMetricString := convertMetricFamiliesIntoTextString(newDeltaRatioMetrics)
				newDeltaRatioMetricStringTotal += newDeltaRatioMetricString
			}
		}

		oldPrometheusMetricString = convertMetricFamiliesIntoTextString(newPrometheusMetrics) + newRateMetricStringTotal + newAvgMetricStringTotal + newRatioMetricStringTotal + newDeltaRatioMetricStringTotal
		// set current to old to prepare new collection in next for loop
		oldPrometheusMetrics = newPrometheusMetrics
	}
}

func getPrometheusMetrics(annotations map[string]string) []*dto.MetricFamily {
	//get prometheus url
	prometheusPort := annotations["prometheus.io/port"]
	if prometheusPort == "" {
		log.Fatalf("\"prometheus.io/port\" can not be empty.")
	}

	prometheusPath := annotations["prometheus.io/path"]
	if prometheusPath == "" {
		prometheusPath = "/metrics"
		log.Infof("\"prometheus.io/path\" is empty, set to default \"/metrics\" for prometheus path.")
	}

	prometheusUrl := getPrometheusUrl(prometheusPort, prometheusPath)

	resp, errGetProm := http.Get(prometheusUrl)
	if errGetProm != nil {
		log.Fatalf("Error scraping prometheus endpoint")
	}
	if resp.ContentLength == 0 {
		log.Warnf("No prometheus metric from %v", prometheusUrl)
	}
	defer resp.Body.Close()
	respBody, errRead := ioutil.ReadAll(resp.Body)
	if errRead != nil {
		log.Fatalf("Error reading response body")
	}
	result, errParse := parsePrometheusMetricsToMetricFamilies(string(respBody))
	if errParse != nil {
		log.Fatalf("Error parsing prometheus metrics to metric families")
	}
	return result
}

func pushPrometheusMetricsString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, oldPrometheusMetricString) // send data to client side
}

func getPodAnnotations(namespace string, podName string) (map[string]string, error) {
	annotations := map[string]string{}
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Failed to create in-cluster config")
	}
	// creates the clientSet
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to creates the clientSet")
	}

	podGet, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Errorf("Pod %v not found in namespace %v.", podName, namespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Errorf("Error getting pod %v in namespace %v: %v", podName, namespace, statusError.ErrStatus.Message)
	} else {
		log.Infof("Found pod %v in namespace %v", podName, namespace)
		annotations = podGet.Annotations
	}
	return annotations, err
}
