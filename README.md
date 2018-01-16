# Monasca Sidecar
A push-pull metric forwarder bridging Monasca and Prometheus. Monasca-sidecar exists as a side container in the same pod with the target container. It gets the pod name and namespace to read annotations.
From annotations, monasca-sidecar will get the prometheus endpoint and sidecar rules.

## Usage

### Add metric list, query interval and listen port to calculate rate.
Under annotations in helm/templates/deployment.yaml, copy prometheus.io/port value to sidecar/listen-port. This will be the port that sidecar will go scrape prometheus metrics.
Set a new value for prometheus.io/port and this will be the port that sidecar will push the calculated metrics to as well as the port monasca-agent should scrape.

Note: both sidecar container and monasca-agent will use the same prometheus.io/path. Default for prometheus.io/path is "/metrics".

prometheus.io/path + prometheus.io/port: monasca-agent to scrape and sidecar to push

prometheus.io/path + sidecar/listen-port: sidecar to scrape

```
prometheus.io/path: "/support/metrics"
prometheus.io/port: "9999"
prometheus.io/scrape: "true"
sidecar/query-interval: "30"
sidecar/listen-port: "5556"
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
```

### Add image information, resource and etc for sidecar container.
In values.yaml

```
sidecar_container:
  log_level: warn
  image:
    repository: 537391133114.dkr.ecr.us-west-1.amazonaws.com/staging/monasca/monasca-sidecar
    tag: 0.0.0-fafad16aec4039 
    pullPolicy: Always
  resources:
    requests:
      memory: 128Mi
      cpu: 50m
    limits:
      memory: 256Mi
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
