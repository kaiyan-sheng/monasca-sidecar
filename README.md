# Monasca Sidecar
A push-pull metric forwarder bridging Monasca and Prometheus.

## Usage
1. Expose pod name and namespace from environment variables
In helm/templates/deployment.yaml
```
        - name: SIDECAR_POD_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.name
        - name: SIDECAR_POD_NAME
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace
```

2. Add metric list, query interval and listen port to calculate rate
In helm/templates/deployment.yaml
```
        sidecar/metric-names: "request_total_time,go_gc_duration_seconds,request_count"
        sidecar/query-interval: "30"
        sidecar/listen-port: "9999"
```

3. Add another container into deployment.yaml
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
```

4. Add values for sidecar container
In values.yaml
```
sidecar_container:
  enabled: true
  image:
    repository: monasca/monasca-sidecar
    tag: latest
    pullPolicy: Always
  resources:
    requests:
      memory: 128Mi
      cpu: 50m
    limits:
      memory: 256Mi
      cpu: 100m
```
