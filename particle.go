package main

import (
	"encoding/json"
	"github.com/r3labs/sse/v2"
	"time"
)

const PARTICLE_EVENT_URL = "https://api.particle.io/v1/devices/events"

type ParticleEvent struct {
	Data      string    `json:"data"`
	TTL       int       `json:"ttl"`
	Published time.Time `json:"published_at"`
	CoreID    string    `json:"coreid"`
}

func MonitorParticle(token *GrainfatherParticleToken, res chan ParticleEvent) {
	var particleUrl = PARTICLE_EVENT_URL + "?access_token=" + token.AccessToken

	client := sse.NewClient(particleUrl)
	events := make(chan *sse.Event)

	err := client.SubscribeChanRaw(events)
	if err != nil {
		panic(err)
	}
	for i := 0; i < 5; i++ {
		var event ParticleEvent
		msg := <-events
		if msg == nil {
			continue
		}

		err = json.Unmarshal(msg.Data[:], &event)
		if err != nil {
			continue
		}
		res <- event
	}
	client.Unsubscribe(events)
}
