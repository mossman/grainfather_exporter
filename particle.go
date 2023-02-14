package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/r3labs/sse/v2"
)

const PARTICLE_EVENT_URL = "https://api.particle.io/v1/devices/events"

type ParticleEvent struct {
	Data      string    `json:"data"`
	TTL       int       `json:"ttl"`
	Published time.Time `json:"published_at"`
	CoreID    string    `json:"coreid"`
}

type ParticleClient struct {
	client *sse.Client
}

func NewParticleClient(token *GrainfatherParticleToken) *ParticleClient {
	var particleUrl = PARTICLE_EVENT_URL + "?access_token=" + token.AccessToken
	client := sse.NewClient(particleUrl)
	return &ParticleClient{client: client}
}

func (c *ParticleClient) Listen(ch chan<- ParticleEvent) {
	events := make(chan *sse.Event)

	err := c.client.SubscribeChanRaw(events)
	if err != nil {
		panic(err)
	}
	for {
		var event ParticleEvent
		log.Println("Waiting event from subscription")
		msg := <-events
		if msg == nil {
			log.Println("Empty message")
			continue
		}

		log.Printf("Msg received %s", &msg.Data)

		err = json.Unmarshal(msg.Data[:], &event)
		if err != nil {
			log.Println("Unmarshal failed")
			continue
		}
		log.Printf("Event received %s", &event)
		ch <- event
	}
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
		res <- event
	}
	client.Unsubscribe(events)
	log.Println("Unsubscribed")
}
