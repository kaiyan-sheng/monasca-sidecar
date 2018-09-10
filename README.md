# Monasca Sidecar
Monasca sidecar is a metric transformer and forwarder bridging Monasca and Prometheus. 
Prometheus provides a functional expression language that lets the user select and aggregate time series data in real time. 
Since monasca does not support prometheus query language, monasca-sidecar is introduced to do similar calculations to 
give user more functionalities to get useful metrics. 
For example, we don't have the ability to collect prometheus metrics on request rate directly but we can collect 
request total count and request time prometheus metrics, and use monasca-sidecar to calculate rate on these two metrics to get rate. 

Monasca-sidecar exists as a side container in the same pod with the target container. 
It gets the exposed pod name and namespace from environment variables in order to get annotations. 
From annotations, monasca-sidecar will get sidecar endpoint, prometheus endpoint and sidecar rules. 
This sidecar container will scrape sidecar endpoint to get prometheus metrics, do calculations 
based on the rules and then push new metrics onto the defined prometheus endpoint for monasca-agent to scrape later. 

## Usage

### Add metric list, query interval and listen port to calculate rate.
Under annotations in helm/templates/deployment.yaml, copy prometheus.io/port value to sidecar/port, copy prometheus.io/path to sidecar/path. This sidecar/path and sidecar/port with create an endpoint for sidecar to scrape prometheus metrics.
Set a new value for prometheus.io/port, prometheus.io/path. This new endpoint will be where monasca-sidecar push calculated metrics to, as well as the port monasca-agent should scrape from.

Note:

* Default for prometheus.io/path is "/metrics".
* Default for sidecar/path is "/metrics".
* prometheus.io/path + prometheus.io/port: monasca-agent to scrape and sidecar to push
* sidecar/path + sidecar/port: sidecar to scrape

```
prometheus.io/path: "/metrics"
prometheus.io/port: "9999"
prometheus.io/scrape: "true"
sidecar/query-interval: "30"
sidecar/port: "5556"
sidecar/path: "/support/metrics"
sidecar/rules: |
  - metricName: request_ratio
    function: ratio
    parameters:
      numerator: request_total_time
      denominator: request_count
  - metricName: request_delta_ratio
    function: deltaRatio
    parameters:
      numerator: request_total_time
      denominator: request_count
  - metricName: request_time_avg
    function: avg
    parameters:
      name: request_total_time
  - metricName: request_count_rate
    function: rate
    parameters:
      name: request_count
  - metricName: request_failure_count_delta
      function: delta
      parameters:
        name: request_failure_count
```

### Add sidecar container into deployment.yaml and expose pod name and namespace from environment variables.
In helm/templates/deployment.yaml

```
      - name: {{ template "name" . }}-sidecar-container
        image: "{{ .Values.sidecar_container.image.repository }}:{{ .Values.sidecar_container.image.tag }}"
        imagePullPolicy: {{ .Values.sidecar_container.image.pullPolicy }}
        resources:
{{ toYaml .Values.sidecar_container.resources | indent 10 }}
        ports:
          - containerPort: 9999
            name: scrape-sidecar
        env:
        - name: SIDECAR_POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
        - name: SIDECAR_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: LOG_LEVEL
          value: {{ .Values.sidecar_container.log_level | quote }}
        - name: RETRY_COUNT
          value: {{ .Values.sidecar_container.retry_count | quote }}
        - name: RETRY_DELAY
          value: {{ .Values.sidecar_container.retry_delay | quote }}
```

### Add image information, resource and etc for sidecar container.
In values.yaml

```
sidecar_container:
  log_level: warn
  retry_count: 5
  retry_delay: 10.0
  image:
    repository: monasca/monasca-sidecar
    tag: 0.0.0-6ac7697938efc1
    pullPolicy: Always
  resources:
    requests:
      memory: 32Mi
      cpu: 50m
    limits:
      memory: 64Mi
      cpu: 100m
```

## Support Functions

### ratio

```
ratio = numerator / denominator
```

### deltaRatio

```
deltaRatio = (numeratorNew - numeratorOld) / (denominatorNew - denominatorOld)
```

### avg

```
avg = (metricValueNew + metricValueOld) / 2
```

### rate

```
rate = (metricValueNew - metricValueOld) / queryInterval
```

### delta

```
delta = metricValueNew - metricValueOld
```
