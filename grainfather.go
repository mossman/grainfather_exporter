package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const GRAINFATHER_AUTH_URL = "https://community.grainfather.com/api/auth/login"
const GRAINFATHER_TOKENS_URL = "https://community.grainfather.com/api/particle/tokens"

type GrainfatherAuth struct {
	Username string `json:"email"`
	Password string `json:"password"`
}

type GrainfatherSession struct {
	ApiToken string `json:"api_token"`
}

type GrainfatherParticleToken struct {
	AccessToken string    `json:"access_token"`
	Expires     time.Time `json:"expires_at"`
}

func AuthenticateGrainfather(username string, password string) (*GrainfatherSession, error) {
	var session GrainfatherSession
	auth := &GrainfatherAuth{Username: username, Password: password}
	b, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", GRAINFATHER_AUTH_URL, bytes.NewBuffer(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(body, &session)
		if err != nil {
			panic(err)
		}
	}
	return &session, nil
}

func GetParticleToken(session *GrainfatherSession) (*GrainfatherParticleToken, error) {
	var tokens []GrainfatherParticleToken
	req, err := http.NewRequest("GET", GRAINFATHER_TOKENS_URL, nil)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+session.ApiToken)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(body, &tokens)
		if err != nil {
			return nil, err
		}
		if len(tokens) == 0 {
			return nil, errors.New("No device token found")
		}
		return &tokens[0], nil
	}
	log.Fatalf("Fail %v", resp)
	return nil, errors.New("Unable to get device token")
}

func ParseConicalFermenterTemp(data string) (float64, float64, error) {
	parts := strings.Split(data, ",")
	if len(parts) < 2 {
		return 0, 0, errors.New("Unable to get temperature")
	}
	temp, err := strconv.ParseFloat(parts[0], 32)
	if err != nil {
		return 0, 0, err
	}
	target, err := strconv.ParseFloat(parts[1], 32)
	if err != nil {
		return 0, 0, err
	}
	return temp, target, nil
}
