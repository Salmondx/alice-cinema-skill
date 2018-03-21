package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type Session struct {
	New       bool   `json:"new"`
	SessionID string `json:"session_id"`
	MessageID int    `json:"message_id"`
	SkillID   string `json:"skill_id"`
	UserID    string `json:"user_id"`
}

type AliceRequest struct {
	Meta struct {
		Locale   string `json:"locale"`
		Timezone string `json:"timezone"`
		ClientID string `json:"client_id"`
	} `json:"meta"`
	Request struct {
		Type   string `json:"type"`
		Markup struct {
			DangerousContext bool `json:"dangerous_context"`
		} `json:"markup"`
		Command           string `json:"command"`
		OriginalUtterance string `json:"original_utterance"`
		Payload           struct {
		} `json:"payload"`
	} `json:"request"`
	Session Session `json:"session"`
	Version string  `json:"version"`
}

type Button struct {
	Title string `json:"title"`
	URL   string `json:"url,omitempty"`
	Hide  bool   `json:"hide"`
}

type AliceResponse struct {
	Version  string  `json:"version"`
	Session  Session `json:"session"`
	Response struct {
		Text       string   `json:"text"`
		Tts        string   `json:"tts"`
		Buttons    []Button `json:"buttons"`
		EndSession bool     `json:"end_session"`
	} `json:"response"`
}

func main() {
	dynamoStorage, err := NewDynamoStorage()
	if err != nil {
		log.Fatal("[ERROR] Failed to init a dynamostorage: %v", err)
	}
	processor := NewProcessor(dynamoStorage)
	http.HandleFunc("/dialog", handler(processor))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("healthy"))
	})
	log.Printf("[INFO] Starting server on port 5000")
	log.Fatal(http.ListenAndServe(":5000", nil))
}

func handler(processor *MessageProcessor) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			log.Printf("[WARN] Wrong method received: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var aliceRequest AliceRequest
		if err := json.NewDecoder(r.Body).Decode(&aliceRequest); err != nil {
			log.Printf("[WARN] Wrong request received: %s", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		userID := aliceRequest.Session.UserID
		if userID == "" {
			log.Printf("[WARN] Request without userID received")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err := time.LoadLocation(aliceRequest.Meta.Timezone)
		if err != nil {
			log.Printf("[WARN] Request with wrong timezone received: %s", aliceRequest.Meta.Timezone)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := processor.Process(&aliceRequest)
		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}
