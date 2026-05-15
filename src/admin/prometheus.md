# Prometheus

To enable <a href="https://prometheus.io/" target="_blank">Prometheus</a> support, add this section to your `config.toml`:

```toml
[observability]
prometheus = true
```

and add this job:

```yaml
- job_name: 'archivetube'
  static_configs:
    - targets: ['archivetube:8080']
  scrape_interval: 1m
```

> [!WARNING]
> There is no auth on top of `/metrics`, which means those metrics are public. If you want to add auth, you need to use a reverse proxy.
> <br><br>
> See <a href="https://www.robustperception.io/adding-basic-auth-to-prometheus-with-nginx/" target="_blank">'Adding Basic Auth to Prometheus with Nginx' on Robust Perception</a>

## Metrics

| Metric name                       | Description                               | Type  |
| --------------------------------- | ----------------------------------------- | ----- |
| archivetube_archive_size          | Total bytes of archived videos            | Gauge |
| archivetube_archived_videos_total | Total number of archived videos           | Gauge |
| archivetube_channels_total        | Total number of channels                  | Gauge |
| archivetube_queue_size            | Current queue size (pending + processing) | Gauge |
