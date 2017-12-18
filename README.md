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

