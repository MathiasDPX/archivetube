package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	queueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "archivetube_queue_size",
			Help: "Current queue size (pending + processing)",
		},
	)

	archivedVideosTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "archivetube_archived_videos_total",
			Help: "Total number of archived videos",
		},
	)

	archiveSizeTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "archivetube_archive_size",
			Help: "Total bytes of archived videos",
		},
	)

	videosTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "archivetube_videos_total",
			Help: "Total number of videos in the database",
		},
	)

	channelsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "archivetube_channels_total",
			Help: "Total number of channels in the database",
		},
	)
)

func Handler() http.Handler {
	return promhttp.Handler()
}

func SetQueueSize(n int) {
	if n < 0 {
		n = 0
	}

	queueSize.Set(float64(n))
}

func IncArchivedVideos() {
	archivedVideosTotal.Inc()
}

func AddArchiveSizeBytes(n int64) {
	if n > 0 {
		archiveSizeTotal.Add(float64(n))
	}
}

func SetVideosTotal(n int) {
	videosTotal.Set(float64(n))
}

func SetChannelsTotal(n int) {
	channelsTotal.Set(float64(n))
}
