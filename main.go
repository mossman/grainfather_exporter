package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/alecthomas/kong"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
)

type Context struct {
}

type AuthCmd struct {
	Username string `required help:"Username" env:"GRAINFATHER_USERNAME"`
	Password string `required help:"Password" env:"GRAINFATHER_PASSWORD"`
}

type ParticleCmd struct {
	Token string `required help:"Token" env:"PARTICLE_TOKEN"`
}

type PrometheusCmd struct {
	ListenAddress string `help:"Listen address" default:":9400"`
	Username      string `required help:"Username" env:"GRAINFATHER_USERNAME"`
	Password      string `required help:"Password" env:"GRAINFATHER_PASSWORD"`
}

const (
	namespace = "grainfather"
)

type Exporter struct {
	temperature *prometheus.Desc
	target      *prometheus.Desc
}

type GrainFatherStatus struct {
	temperature float64
	target      float64
}

var grainFatherStatus = &GrainFatherStatus{}

func NewExporter() *Exporter {
	return &Exporter{
		temperature: prometheus.NewDesc(
			"temperature",
			"Fermenter temperature",
			nil,
			nil),
		target: prometheus.NewDesc(
			"target",
			"Fermenter target",
			nil,
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

func RenewParticleToken(Username string, Password string) (*GrainfatherParticleToken, error) {
	session, err := AuthenticateGrainfather(Username, Password)
	if err != nil {
		return nil, err
	}
	token, err := GetParticleToken(session)
	if err != nil {
		panic(err)
	}
	return token, nil
}

func getConicalFermenterTemp(token *GrainfatherParticleToken) float64 {
	devices := GetParticleDevices(token)

	eventchan := make(chan Measurement)
	log.Print("Starting monitor")
	go GetMeasurementFromParticle(token, eventchan, &devices[0])

	log.Print("Waiting event")

	ev := <-eventchan
	return ev.Temperature
}

func (p *ParticleCmd) Run(ctx *Context) error {
	token := GrainfatherParticleToken{AccessToken: p.Token}

	temp := getConicalFermenterTemp(&token)
	fmt.Println(temp)
	return nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.temperature
	ch <- e.target
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.temperature, prometheus.GaugeValue, float64(grainFatherStatus.temperature))
	ch <- prometheus.MustNewConstMetric(e.target, prometheus.GaugeValue, float64(grainFatherStatus.target))
}

func (p *PrometheusCmd) Run(ctx *Context) error {
	wg := new(sync.WaitGroup)
	wg.Add(1)

	token, err := RenewParticleToken(p.Username, p.Password)
	if err != nil {
		panic(err)
	}
	exporter := NewExporter()

	devices := GetParticleDevices(token)

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
	go func() {
		if err := http.ListenAndServe(p.ListenAddress, nil); err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
			wg.Done()
		}
	}()
	for {
		log.Print("Measuring...")
		if time.Now().After(token.Expires) {
			token, err = RenewParticleToken(p.Username, p.Password)
			if err != nil {
				panic(err)
			}
		}
		StartMonitorActivity(token, devices[0].Id, 2)
		ch := make(chan Measurement)
		event, err := GetMeasurementFromParticle(token, ch, &devices[0])
		if err != nil {
			panic(err)
		}
		temp := event.Temperature
		target := event.Target
		if err != nil {
			log.Printf("Error from particle: %v", err)
		}
		log.Printf("Temp: %v Target: %v", temp, target)
		grainFatherStatus.temperature = temp
		grainFatherStatus.target = target
		time.Sleep(1 * time.Minute)
	}

	wg.Wait()
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
