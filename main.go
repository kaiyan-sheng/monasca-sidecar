// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"fmt"
	prometheusClient "github.com/prometheus/client_model/go"
	log "github.hpe.com/kronos/kelog"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"net/http"
	"os"
	"strings"
	"time"
)

var oldPrometheusMetricString = ``

func main() {
	// set log level
	setLogLevel()
	// get annotations from pod kube config
	annotations := getPodAnnotations()
	// get Prometheus url
	prometheusUrl, succeedFlag := getPrometheusUrl(annotations)
	log.Infof("Sidecar gets prometheus metrics from URL = %v", prometheusUrl)

	if !succeedFlag {
		log.Fatalf("Errror getting prometheus URL.")
	}
	// get rules from annotations
	sidecarRulesString, queryInterval, listenPortPath := getSidecarRulesFromAnnotations(annotations)
	log.Infof("Sidecar pushes new prometheus metric to %v", listenPortPath)

	sidecarRules := parseYamlSidecarRules(sidecarRulesString)
	// get prometheus url and prometheus metric response body
	oldPrometheusMetrics := getPrometheusMetrics(prometheusUrl)
	oldPrometheusMetricString = convertMetricFamiliesIntoTextString(oldPrometheusMetrics)

	// start web server
	http.HandleFunc("/", pushPrometheusMetricsString) // set router
	go http.ListenAndServe(":"+listenPortPath, nil)   // set listen port

	// Infinite for loop to scrape prometheus metrics and calculate rate every 30 seconds
	for {
		newRateMetricStringTotal := ``
		newAvgMetricStringTotal := ``
		newRatioMetricStringTotal := ``
		newDeltaRatioMetricStringTotal := ``

		// sleep for 30 seconds or how long queryInterval is
		time.Sleep(time.Second * time.Duration(queryInterval))

		// get a new set of prometheus metrics
		newPrometheusMetrics := getPrometheusMetrics(prometheusUrl)

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
			default:
				log.Errorf("Rule %v with invalid function %v", rule.Name, rule.Function)
			}
		}

		oldPrometheusMetricString = convertMetricFamiliesIntoTextString(newPrometheusMetrics) + newRateMetricStringTotal + newAvgMetricStringTotal + newRatioMetricStringTotal + newDeltaRatioMetricStringTotal
		// set current to old to prepare new collection in next for loop
		oldPrometheusMetrics = newPrometheusMetrics
	}
}

func getPrometheusUrl(annotations map[string]string) (string, bool) {
	//get prometheus url
	prometheusPort := annotations["sidecar/listen-port"]
	if prometheusPort == "" {
		log.Errorf("\"sidecar/listen-port\" can not be empty.")
		return "", false
	}

	prometheusPath := annotations["prometheus.io/path"]
	if prometheusPath == "" {
		prometheusPath = "/metrics"
		log.Infof("\"prometheus.io/path\" is empty, set to default \"/metrics\" for prometheus path.")
	}

	// check annotations
	scrape := annotations["prometheus.io/scrape"]
	if scrape != "true" {
		log.Errorf("Scrape prometheus metrics is not enabled. Please enable prometheus.io/scrape in annotations first.")
		return "", false
	}

	prefix := "http://localhost"
	if prometheusPath == "/" {
		prometheusUrl := prefix + ":" + prometheusPort
		return prometheusUrl, true
	}
	if strings.HasSuffix(prometheusPath, "/") {
		prometheusPath := prometheusPath[:(len(prometheusPath) - 1)]
		prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
		return prometheusUrl, true
	}
	prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
	return prometheusUrl, true
}

func getPrometheusMetrics(prometheusUrl string) []*prometheusClient.MetricFamily {
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

func getPodAnnotations() map[string]string {
	//get namespace and pod name from environment variables
	podNamespace, ok := os.LookupEnv("SIDECAR_POD_NAMESPACE")
	if !ok {
		log.Fatalf("%s not set\n", "SIDECAR_POD_NAMESPACE")
	}

	podName, ok := os.LookupEnv("SIDECAR_POD_NAME")
	if !ok {
		log.Fatalf("%s not set\n", "SIDECAR_POD_NAME")
	}

	// get annotations
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

	podGet, err := clientSet.CoreV1().Pods(podNamespace).Get(podName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Fatalf("Pod %v not found in namespace %v.", podName, podNamespace)
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Fatalf("Error getting pod %v in namespace %v: %v", podName, podNamespace, statusError.ErrStatus.Message)
	} else {
		log.Infof("Found pod %v in namespace %v", podName, podNamespace)
		annotations = podGet.Annotations
	}
	return annotations
}

func setLogLevel() {
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
}
