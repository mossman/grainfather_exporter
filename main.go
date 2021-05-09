package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/alecthomas/kong"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

type Context struct {
}

type AuthCmd struct {
	Username string `help:"Username" env:"GRAINFATHER_USERNAME"`
	Password string `help:"Password" env:"GRAINFATHER_PASSWORD"`
}

type ParticleCmd struct {
	Token string `help:"Token" env:"PARTICLE_TOKEN"`
}

type PrometheusCmd struct {
	ListenAddress string `help:"Listen address" default:":9400"`
	Token         string `help:"Token" env:"PARTICLE_TOKEN"`
}

const (
	namespace = "grainfather"
)

type Exporter struct {
	mutex sync.Mutex
	token *GrainfatherParticleToken

	up          *prometheus.Desc
	temperature *prometheus.Desc
}

func NewExporter(token *GrainfatherParticleToken) *Exporter {
	return &Exporter{
		token: token,
		up: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "up"),
			"Could fermenter be reached",
			nil,
			nil),
		temperature: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "", "temperature"),
			"Fermenter temperature",
			[]string{"pluginId", "pluginCategory"},
			nil),
	}
}

func (a *AuthCmd) Run(ctx *Context) error {
	session, err := AuthenticateGrainfather(a.Username, a.Password)
	if err != nil {
		return err
	}
	token, err := GetParticleToken(session)
	if err != nil {
		panic(err)
	}
	fmt.Println(token)
	return nil
}

func getConicalFermenterTemp(token *GrainfatherParticleToken) (float64, error) {
	eventchan := make(chan ParticleEvent)
	go MonitorParticle(token, eventchan)

	ev := <-eventchan
	temp, err := ParseConicalFermenterTemp(ev.Data)
	if err != nil {
		return 0, err
	}
	return temp, err
}

func (p *ParticleCmd) Run(ctx *Context) error {
	token := GrainfatherParticleToken{AccessToken: p.Token}

	temp, err := getConicalFermenterTemp(&token)
	if err != nil {
		panic(err)
	}
	fmt.Println(temp)
	return nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.temperature
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	log.Println("Getting temp")

	temp, err := getConicalFermenterTemp(e.token)
	if err != nil {
		panic(err)
	}

	ch <- prometheus.MustNewConstMetric(e.temperature, prometheus.GaugeValue, float64(temp), "", "")
}

func (p *PrometheusCmd) Run(ctx *Context) error {

	token := GrainfatherParticleToken{AccessToken: p.Token}
	exporter := NewExporter(&token)

	prometheus.MustRegister(exporter)
	prometheus.MustRegister(version.NewCollector("grainfather_exporter"))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
            <head>
            <title>Grainfather Exporter</title>
            </head>
            <body>
            <h1>Grainfather Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	http.Handle("/metrics", promhttp.Handler())
	log.Printf("grainfather_exporter listening on port %v", p.ListenAddress)
	if err := http.ListenAndServe(p.ListenAddress, nil); err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
		return err
	}
	return nil
}

var CLI struct {
	Auth       AuthCmd       `cmd help:"Authenticate to grainfather."`
	Particle   ParticleCmd   `cmd help:"Gather events from particle.io"`
	Prometheus PrometheusCmd `cmd help:"Serve prometheus metrics"`
}

func main() {
	ctx := kong.Parse(&CLI)
	err := ctx.Run(&Context{})
	ctx.FatalIfErrorf(err)
}
