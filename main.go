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

type Dimension struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type DimensionList []Dimension

type PrometheusMetric struct {
	Name          string        `json:"name"`
	Value         string        `json:"value"`
	Dimensions    DimensionList `json:"dimensions"`
	DimensionHash []byte        `json:"hashcode"`
}

var oldRateMetricString = ``

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
		podNamespace = "ms"
		// os.Exit(1)
	}

	podName, ok := os.LookupEnv("SIDECAR_POD_NAME")
	if !ok {
		log.Errorf("%s not set\n", "SIDECAR_POD_NAME")
		// os.Exit(1)
		podName = "ms-api-api-4195849451-1qm50"
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
	fmt.Println("*************")
	fmt.Println(rules)
	fmt.Println("*************")

	sidecarRules := parseYamlSidecarRules(rules)
	// get all metric names from sidecar rules into metricNameArray
	// metricNameArray := getMetricNamesFromRules(sidecarRules)

	queryInterval, err := strconv.ParseFloat(queryIntervalString, 64)
	if queryInterval <= 0.0 || err != nil {
		log.Warnf("Error converting \"sidecar/query-interval\". Set queryInterval to default 30.0 seconds.")
		queryInterval = 30.0
	}

	// get prometheus url and prometheus metric response body
	oldPrometheusMetrics := getPrometheusMetrics(annotations)

	// start web server
	http.HandleFunc("/", pushPrometheusMetricsString) // set router
	go http.ListenAndServe(":"+listenPort, nil)       // set listen port

	// Infinite for loop to scrape prometheus metrics and calculate rate every 30 seconds
	for {
		//newRateMetricString := ``
		//newAvgMetricString := ``
		//newRatioMetricString := ``
		// sleep for 30 seconds or how long queryInterval is
		time.Sleep(time.Second * time.Duration(queryInterval))

		// get a new set of prometheus metrics
		newPrometheusMetrics := getPrometheusMetrics(annotations)
		fmt.Println("******** NEW *******")
		for _, mf := range newPrometheusMetrics {
			fmt.Println(mf)
		}

		// calculate by each sidecar rule
		for _, rule := range sidecarRules {
			if rule.Function == "rate" {
				newRateMetrics := calculateRate(newPrometheusMetrics, oldPrometheusMetrics, queryInterval, rule)
				newRateMetricString := convertMetricFamiliesIntoTextString(newRateMetrics)
				fmt.Println(newRateMetricString)
				fmt.Println("********************")
			}
			//} else if rule.Function == "avg" {
			//	newAvgMetricString += calculateAvg(newPrometheusMetrics, oldPrometheusMetrics, rule)
			//	fmt.Println(newAvgMetricString)
			//	fmt.Println("********************")
			//} else if rule.Function == "ratio" {
			//	newRatioMetricString += calculateRatio(newPrometheusMetrics, rule)
			//	fmt.Println(newAvgMetricString)
			//	fmt.Println("********************")
			//}
		}

		// set current to old to prepare new collection in next for loop
		oldPrometheusMetrics = newPrometheusMetrics
	}
}

func responseBodyToStructure(respBody string, metricName string, prometheusMetrics []PrometheusMetric) []PrometheusMetric {
	// Find metric name and parse the response body string
	if !strings.Contains(respBody, metricName) {
		log.Infof("Prometheus metrics does not include %v", metricName)
		return prometheusMetrics
	}

	splitWithName := strings.Split(respBody, "# HELP "+metricName)
	metricString := strings.Split(splitWithName[1], "# HELP")[0]

	// Convert a string into structure
	metricStringLines := strings.Split(metricString, "\n")
	// Conver each line
	for _, i := range metricStringLines[2:] {
		metricSplit := strings.Split(i, " ")
		if len(metricSplit) > 1 {
			metricDimensions := []Dimension{}
			//get metric value
			metricValue := metricSplit[1]
			//get metric name
			if strings.ContainsAny(string(i), "{") {
				iMetricName := strings.Split(string(i), "{")[0]
				// get dimensions
				dimensions := stringBetween(string(i), "{", "}")
				splitDims := strings.Split(dimensions, ",")
				for _, d := range splitDims {
					split_each_dim := strings.Split(d, "=")
					dim := Dimension{Key: split_each_dim[0], Value: split_each_dim[1]}
					metricDimensions = append(metricDimensions, dim)
				}
				// sortedMetricDimensions := sortDimensionsByKeys(metricDimensions)
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

	resp, err := http.Get(prometheusUrl)
	if err != nil {
		log.Fatalf("Error scraping prometheus endpoint")
	}
	if resp.ContentLength == 0 {
		log.Warnf("No prometheus metric from %v", prometheusUrl)
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	result, err := parsePrometheusMetricsToMetricFamilies(string(respBody))
	return result
}

func pushPrometheusMetricsString(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, oldRateMetricString) // send data to client side
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
	} else if err != nil {
	} else {
		log.Infof("Found pod %v in namespace %v", podName, namespace)
		annotations = podGet.Annotations
	}
	return annotations, err
}

func getMetricNamesFromRules(sidecarRules []SidecarRule) []string {
	metricNames := []string{}
	for _, rule := range sidecarRules {
		if rule.Function == "rate" {
			metricNames = append(metricNames, rule.Parameters["name"])
		} else if rule.Function == "avg" {
			metricNames = append(metricNames, rule.Parameters["name"])
		} else if rule.Function == "ratio" {
			metricNames = append(metricNames, rule.Parameters["numerator"])
			metricNames = append(metricNames, rule.Parameters["demoninator"])
		}
	}
	metricNameArray := removeDuplicates(metricNames)
	return metricNameArray
}
