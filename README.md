# Monasca Sidecar
A push-pull metric forwarder bridging Monasca and Prometheus.

## Usage
1. Add metric list, query interval and listen port to calculate rate. 
Under annotations in helm/templates/deployment.yaml, add:
```
        sidecar/metric-names: "request_total_time,go_gc_duration_seconds,request_count"
        sidecar/query-interval: "30"
        sidecar/listen-port: "9999"
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
