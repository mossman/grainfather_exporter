package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/r3labs/sse/v2"
)

const PARTICLE_DEVICE_URL = "https://api.particle.io/v1/devices/"

type Measurement struct {
	Temperature float64 `json:"temp"`
	Target      float64 `json:"target"`
	Heating     bool    `json:"heatStatus"`
	Cooling     bool    `json:"coolStatus"`
}

type GrainFatherData struct {
	Data Measurement `json:"data"`
}

type ParticleEvent struct {
	Data      string    `json:"data"`
	TTL       int       `json:"ttl"`
	Published time.Time `json:"published_at"`
	CoreID    string    `json:"coreid"`
}

type ParticleDevice struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func GetParticleDevices(token *GrainfatherParticleToken) []ParticleDevice {
	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}
	req, err := http.NewRequest("GET", PARTICLE_DEVICE_URL, nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var devices []ParticleDevice
	err = json.Unmarshal(body, &devices)
	if err != nil {
		log.Fatal(err)
	}
	return devices
}

func StartMonitorActivity(token *GrainfatherParticleToken, DeviceID string, duration int) {
	var particleUrl = PARTICLE_DEVICE_URL + DeviceID + "/highActivity"

	data := url.Values{}
	data.Set("args", strconv.Itoa(duration))

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}
	req, err := http.NewRequest("POST", particleUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	if res.Body != nil {
		defer res.Body.Close()
	}
}

func GetMeasurementFromParticle(token *GrainfatherParticleToken, res chan Measurement, device *ParticleDevice) (*Measurement, error) {
	var particleUrl = PARTICLE_DEVICE_URL + "events"
	client := sse.NewClient(particleUrl)
	client.Headers["Authorization"] = "Bearer " + token.AccessToken
	events := make(chan *sse.Event)

	err := client.SubscribeChanRaw(events)
	if err != nil {
		panic(err)
	}
	defer client.Unsubscribe(events)

	for i := 0; i < 5; i++ {
		var event ParticleEvent
		log.Println("Waiting event from subscription")
		msg := <-events
		if msg == nil {
			log.Println("Empty message")
			continue
		}

		log.Println(string(msg.Data))
		err = json.Unmarshal(msg.Data[:], &event)
		if err != nil {
			log.Printf("Unmarshal failed %v", err)
			continue
		}
		log.Printf("Event received %v", &event)
		var grainfatherdata GrainFatherData
		if event.CoreID == device.Id {
			err = json.Unmarshal([]byte(event.Data), &grainfatherdata)
			if err != nil {
				log.Printf("Unmarshal payload failed %v", err)
				continue
			}

			return &grainfatherdata.Data, nil
		}
	}
	return nil, errors.New("No event received")
}
