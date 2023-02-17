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
	var particleUrl = PARTICLE_DEVICE_URL + "?access_token=" + token.AccessToken

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}
	req, err := http.NewRequest("GET", particleUrl, nil)
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
	var particleUrl = PARTICLE_DEVICE_URL + DeviceID + "/highActivity?access_token=" + token.AccessToken

	data := url.Values{}
	data.Set("args", strconv.Itoa(duration))

	client := http.Client{
		Timeout: time.Second * 2, // Timeout after 2 seconds
	}
	req, err := http.NewRequest("POST", particleUrl, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
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

func GetEventFromParticle(token *GrainfatherParticleToken, res chan ParticleEvent, device *ParticleDevice) (*ParticleEvent, error) {
	var particleUrl = PARTICLE_DEVICE_URL + "events?access_token=" + token.AccessToken

	client := sse.NewClient(particleUrl)
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

		err = json.Unmarshal(msg.Data[:], &event)
		if err != nil {
			log.Println("Unmarshal failed")
			continue
		}
		log.Printf("Event received %s", &event)
		if event.CoreID == device.Id {
			return &event, nil
		}
	}
	return nil, errors.New("No event received")
}
