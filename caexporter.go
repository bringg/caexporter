package main

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	up = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "up",
		Help: "is caexporter up",
	})

	lastActivity = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cluster_autoscaler_last_activity",
			Help: "LastProbeTime as reported in cluster-autoscaler-status configmap",
		}, []string{"activity"},
	)
)

type Collector struct {
	kubeClient *kubernetes.Clientset
	regex      *regexp.Regexp
}

func newCollector() *Collector {
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	).ClientConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	log.Info().Msg("creating kubernetes clientset")

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatal().Err(err).Msg("")
	}

	rx := regexp.MustCompile(`(?s)LastProbeTime:\s*(.{39})`)

	c := &Collector{
		kubeClient: clientset,
		regex:      rx,
	}

	return c
}

func main() {
	r := prometheus.NewRegistry()
	r.MustRegister(newCollector())

	http.Handle("/metrics", promhttp.HandlerFor(r, promhttp.HandlerOpts{}))
	log.Info().Msg("Beginning to serve on port :8080")
	log.Fatal().Err(http.ListenAndServe(":8080", nil))
}

// Describe implements prometheus.Collector
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- up.Desc()
	lastActivity.Describe(ch)
}

// Collect implements prometheus.Collector
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	up.Set(1)
	ch <- up

	cm, err := c.kubeClient.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "cluster-autoscaler-status", metav1.GetOptions{})
	if err != nil {
		log.Fatal().Err(err)
	}

	cmData := cm.Data["status"]

	cw := c.regex.FindAllStringSubmatch(cmData, 3)

	res := map[string]string{
		"main":      cw[0][1],
		"scaleUp":   cw[1][1],
		"scaleDown": cw[2][1],
	}

	for act, val := range res {
		layout := "2006-01-02 15:04:05 -0700 MST"
		activityTime, err := time.Parse(layout, val)
		if err != nil {
			log.Fatal().Err(err)
		}

		lastActivity.With(prometheus.Labels{"activity": act}).Set(float64(activityTime.Unix()))

		ch <- lastActivity.With(prometheus.Labels{"activity": act})
	}
}
