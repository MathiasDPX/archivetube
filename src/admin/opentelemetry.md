# Open Telemetry

You can enable <a href="https://opentelemetry.io/" target="_blank">OpenTelemetry</a> by specifying an endpoint in the config:

```toml
[observability]
otel_exporter_otlp_endpoint="http://localhost:4318"
```

If this variable is empty or absent, OpenTelemetry will be disabled by default