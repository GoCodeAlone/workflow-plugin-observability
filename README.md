# workflow-plugin-observability

External Workflow plugin for OTel-first telemetry collection.

## Modules

- `observability.telemetry` receives neutral metrics, logs, and span events from Workflow's host-side telemetry bridge.
- `observability.collector` validates collector intent, renders OTel Collector config, and returns app environment wiring.

## Minimal External Collector

```yaml
modules:
  - name: telemetry
    type: observability.telemetry
    config:
      serviceName: checkout-api
      environment: production

  - name: checkout-api
    type: observability.collector
    config:
      distribution: external
      topology: external
      signals: [traces, metrics, logs]
      endpoint: https://otel-collector.example.com:4318
```

OTel/OTLP is the preferred path for traces, metrics, and logs. Direct Grafana, Prometheus, Loki, and Datadog integrations are planned for backend-specific resources and runtime choices where OTel is not the best fit, such as dashboards, monitors, Loki push, Prometheus scrape/remote-write helpers, and Datadog Agent/DDOT deployment.

## Custom Telemetry

Applications and plugins emit telemetry by satisfying Workflow SDK interfaces, not by importing this plugin. Metric names belong to the implementing application or domain module; this plugin records and routes the names it receives without imposing prefixes.

## Rollback

Remove `observability.telemetry` and `observability.collector` from Workflow YAML and redeploy. Generated collector resources are labeled with `workflow.gocodealone.io/managed-by: workflow-plugin-observability` so provider adapters can roll back only resources owned by this plugin.
