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

type AliceResponse struct {
	Version  string  `json:"version"`
	Session  Session `json:"session"`
	Response struct {
		Text    string `json:"text"`
		Tts     string `json:"tts"`
		Buttons []struct {
			Title   string `json:"title"`
			Payload struct {
			} `json:"payload"`
			URL  string `json:"url"`
			Hide bool   `json:"hide"`
		} `json:"buttons"`
		EndSession bool `json:"end_session"`
	} `json:"response"`
}

func main() {
	// fmt.Println(GetShowtimes("излом времени", "москва", "теплый стан"))
	// fmt.Println(GetRamblerShowtimes("черная пантера", "", "ясенево"))

	// loc, err := GetUserLocation("в москве симферопольский проспект")
	// if err != nil {
	// 	fmt.Println(err)
	// 	os.Exit(1)
	// }
	// fmt.Println(loc.Subway + ": " + loc.City)

	processor := NewProcessor(NewStorage())
	http.HandleFunc("/dialog", handler(processor))
	log.Printf("[INFO] Starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(processor *MessageProcessor) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var aliceRequest AliceRequest
		if err := json.NewDecoder(r.Body).Decode(&aliceRequest); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		userID := aliceRequest.Session.UserID
		if userID == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, err := time.LoadLocation(aliceRequest.Meta.Timezone)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		response := processor.Process(&aliceRequest)
		json.NewEncoder(w).Encode(response)
	}
}
