# Monasca Sidecar
A push-pull metric forwarder bridging Monasca and Prometheus.

## Usage
1. Add metric list, query interval and listen port to calculate rate. 
Under annotations in helm/templates/deployment.yaml, add:

```
sidecar/query-interval: "30"
sidecar/listen-port: "9999"
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

2. Add sidecar container into deployment.yaml and expose pod name and namespace from environment variables. 
In helm/templates/deployment.yaml, add:

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

3. Add image information, resource and etc for sidecar container. 
In values.yaml, add:

```
sidecar_container:
  log_level: info
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

1. ratio

```
ratio = numerator / denominator
```

2. deltaRatio

```
deltaRatio = (numeratorNew - numeratorOld) / (denominatorNew - denominatorOld)
```

3. avg

```
avg = (metricValueNew + metricValueOld) / 2
```

4. rate

```
rate = (metricValueNew - metricValueOld) / queryInterval
```
