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
	"strconv"
	"strings"
	"time"
)

func main() {
	// set log level
	setLogLevel()
	// retry to get annotations
	annotations := retryGetAnnotations()
	// get Prometheus url
	prometheusUrl, succeedFlag := getPrometheusUrl(annotations)

	if !succeedFlag {
		log.Fatalf("Errror getting prometheus URL.")
	}
	// get rules from annotations
	sidecarRulesString, queryInterval, listenPort, listenPath := getSidecarRulesFromAnnotations(annotations)
	log.Infof("Sidecar gets prometheus metrics from URL = %v", prometheusUrl)
	log.Infof("Sidecar pushes new prometheus metric to %v", listenPort+listenPath)

	sidecarRules := parseYamlSidecarRules(sidecarRulesString)
	// get prometheus url and prometheus metric response body
	oldPrometheusMetrics := getPrometheusMetrics(prometheusUrl)
	oldPrometheusMetricString := convertMetricFamiliesIntoTextString(oldPrometheusMetrics)

	// start web server
	http.HandleFunc(listenPath, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, oldPrometheusMetricString) // send data to client side
	})
	go http.ListenAndServe(":"+listenPort, nil) // set listen port

	// Infinite for loop to scrape prometheus metrics and calculate rate every 30 seconds
	for {
		newRateMetrics := []*prometheusClient.MetricFamily{}
		newAvgMetrics := []*prometheusClient.MetricFamily{}
		newRatioMetrics := []*prometheusClient.MetricFamily{}
		newDeltaRatioMetrics := []*prometheusClient.MetricFamily{}
		newDeltaMetrics := []*prometheusClient.MetricFamily{}

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
				newRateMetrics = append(newRateMetrics, calculateRate(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, queryInterval, rule)...)
			case "avg":
				newAvgMetrics = append(newAvgMetrics, calculateAvg(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, rule)...)
			case "ratio":
				newRatioMetrics = append(newRatioMetrics, calculateRatio(newPrometheusMetricsWithNoHistogramSummary, rule)...)
			case "deltaRatio":
				newDeltaRatioMetrics = append(newDeltaRatioMetrics, calculateDeltaRatio(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, rule)...)
			case "delta":
				newDeltaMetrics = append(newDeltaMetrics, calculateDelta(newPrometheusMetricsWithNoHistogramSummary, oldPrometheusMetricsWithNoHistogramSummary, rule)...)
			default:
				log.Errorf("Rule %v with invalid function %v", rule.Name, rule.Function)
			}
		}
		oldPrometheusMetricString = convertMetricFamiliesIntoTextString(newPrometheusMetrics) + convertMetricFamiliesIntoTextString(newRateMetrics) + convertMetricFamiliesIntoTextString(newAvgMetrics) + convertMetricFamiliesIntoTextString(newRatioMetrics) + convertMetricFamiliesIntoTextString(newDeltaRatioMetrics) + convertMetricFamiliesIntoTextString(newDeltaMetrics)
		// set current to old to prepare new collection in next for loop
		oldPrometheusMetrics = newPrometheusMetrics
	}
}

func getPrometheusUrl(annotations map[string]string) (string, bool) {
	//get prometheus url
	prometheusPort := annotations["sidecar/port"]
	if prometheusPort == "" {
		log.Errorf("\"sidecar/port\" can not be empty.")
		return "", false
	}

	prometheusPath := annotations["sidecar/path"]
	if prometheusPath == "" {
		prometheusPath = "/metrics"
		log.Infof("\"sidecar/path\" is empty, set to default \"/metrics\" for sidecar path.")
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
		prometheusPath = prometheusPath[:(len(prometheusPath) - 1)]
	}
	prometheusUrl := prefix + ":" + prometheusPort + prometheusPath
	return prometheusUrl, true
}

func getPrometheusMetrics(prometheusUrl string) []*prometheusClient.MetricFamily {
	// http.get prometheus url with retries
	retryCount, retryDelay := getRetryParams()
	resp := &http.Response{}
	for i := 1; i <= retryCount; i++ {
		resp, errGetProm := http.Get(prometheusUrl)
		if errGetProm == nil {
			log.Debugf("Http Get works! resp = ", resp)
			break
		}
		log.Infof("Error scraping prometheus endpoint. Retrying. Sleep %v seconds and retry %v.", retryDelay, i)
		// sleep for 10 seconds or how long retry_delay is
		time.Sleep(time.Second * time.Duration(retryDelay))
		if i == retryCount {
			log.Fatalf("Failed to scrape prometheus endpoint %v with %v times of retries.", prometheusUrl, retryCount)
		}
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
	logLevelEnv := "warn"
	if ok {
		logLevelEnv = val
	}
	logLevel := strings.ToLower(logLevelEnv)
	if logLevel != "" {
		log.Printf("Setting global log level to '%s'", logLevel)
		log.SetLevelString(logLevel)
	}
}

func getRetryParams() (int, float64) {
	retryCount, okCount := os.LookupEnv("RETRY_COUNT")
	retryDelay, okDelay := os.LookupEnv("RETRY_DELAY")
	retryCountEnv := 5
	retryDelayEnv := 10.0
	if okCount {
		retryCountEnvInt, errInt := strconv.Atoi(retryCount)
		if errInt == nil {
			retryCountEnv = retryCountEnvInt
		} else {
			log.Warnf("Error converting RETRY_COUNT to an integer. Set to default RETRY_COUNT=5.")
		}
	}
	if okDelay {
		retryDelayEnvFloat, errFloat := strconv.ParseFloat(retryDelay, 64)
		if errFloat == nil {
			retryDelayEnv = retryDelayEnvFloat
		} else {
			log.Warnf("Error converting RETRY_DELAY to a float. Set to default RETRY_DELAY=10.0.")
		}
	}
	return retryCountEnv, retryDelayEnv
}

func retryGetAnnotations() map[string]string {
	// get retry params
	retryCount, retryDelay := getRetryParams()
	log.Infof("retryCount = ", retryCount)
	log.Infof("retryDelay = ", retryDelay)
	// get annotations from pod kube config
	annotations := map[string]string{}
	for i := 1; i <= retryCount; i++ {
		annotations := getPodAnnotations()
		if _, ok := annotations["sidecar/port"]; ok {
			log.Debugf("Good annotation! annotations = ", annotations)
			return annotations
		}
		log.Infof("Annotation doesn't include all the information that's needed. Sleep %v seconds and retry %v.", retryDelay, i)
		// sleep for 10 seconds or how long retry_delay is
		time.Sleep(time.Second * time.Duration(retryDelay))
	}
	return annotations
}
