package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const namespace = "cluster_autoscaler"

var (
	webListenAddress        = flag.String("web.listen-address", ":8080", "Address to listen on for web interface and telemetry")
	webMetricsPath          = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics")
	collectorRequestTimeout = flag.Int("collector.request-timeout", 10, "Kubernetes API request timeout in seconds")
	logDebug                = flag.Bool("log.debug", false, "sets log level to debug")
	appVersion              = flag.Bool("version", false, "prints the exporter version")

	desc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "", "last_activity"),
		"LastProbeTime as reported in cluster-autoscaler-status configmap",
		[]string{"activity"},
		nil,
	)

	scrapeError = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "scrape_error",
		Help:      "Set to 0 for successful scrape, or 1 otherwise",
	})
)

type Collector struct {
	kubeClient *kubernetes.Clientset
	mu         *sync.Mutex
	regex      *regexp.Regexp
}

func init() {
	prometheus.MustRegister(version.NewCollector(namespace), scrapeError)
}

func newCollector() *Collector {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	log.Info().Msg("Creating kubernetes clientset")

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	// we're looking for lines that look like -
	// LastProbeTime:      2022-09-11 11:21:27.046154458 +0000 UTC m=+25121.073170634
	// but without the m=... part
	rx := regexp.MustCompile(`(?s)LastProbeTime:\s*([\d\-]*\s[\d\:\.]*\s[\+\d]*\sUTC)`)

	c := &Collector{
		kubeClient: clientset,
		mu:         &sync.Mutex{},
		regex:      rx,
	}

	return c
}

// Collect implements prometheus.Collector
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*collectorRequestTimeout*int(time.Second)))
	defer cancel()

	cm, err := c.kubeClient.CoreV1().ConfigMaps("kube-system").Get(ctx, "cluster-autoscaler-status", metav1.GetOptions{})
	if err != nil {
		log.Error().Err(err).Msg("")
		scrapeError.Set(1)
		return
	}

	cmData := cm.Data["status"]

	cw := c.regex.FindAllStringSubmatch(cmData, 3)
	// nil means there's no regex match on the returned data,
	// for example the configmap is still in initialized phase
	if cw == nil {
		log.Info().Msg("Can't match regex pattern on the returned data")
		log.Debug().Msgf("Received configmap status %s, regex pattern is %s", cmData, c.regex.String())
		scrapeError.Set(1)
		return
	}

	res := map[string]string{
		"main":      cw[0][1],
		"scaleUp":   cw[1][1],
		"scaleDown": cw[2][1],
	}

	if len(res) != 3 {
		log.Error().Msg("Couldn't extract required data from the ConfigMap")
		scrapeError.Set(1)
		return
	}

	dateLayout := "2006-01-02 15:04:05 -0700 MST"

	for act, val := range res {
		activityTime, err := time.Parse(dateLayout, val)
		if err != nil {
			log.Error().Err(err).Msg("")
			scrapeError.Set(1)
			return
		}

		metric, err := prometheus.NewConstMetric(desc, prometheus.GaugeValue, float64(activityTime.Unix()), act)
		if err != nil {
			log.Error().Err(err).Msg("")
		}

		scrapeError.Set(0)
		ch <- metric
	}
}

// Describe implements prometheus.Collector
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(c, ch)
}

func main() {
	flag.Parse()

	if *appVersion {
		fmt.Println(version.Version)
		return
	}

	// log configs
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if *logDebug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// adds line location to the caller field of the event
	log.Logger = log.With().Caller().Logger()

	prometheus.MustRegister(newCollector())

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//nolint:errcheck
		w.Write([]byte(`<html>
             <head><title>CAExporter Exporter</title></head>
             <body>
             <h1>CAExporter Exporter</h1>
             <p><a href='` + *webMetricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Info().Msg("Beginning to serve on port " + *webListenAddress)
	log.Fatal().Err(http.ListenAndServe(*webListenAddress, nil)).Msg("")
}
