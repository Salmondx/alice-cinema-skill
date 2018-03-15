package main

import (
	"encoding/json"
	"fmt"
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

	http.HandleFunc("/dialog", handler(NewStorage(), Default()))
	log.Printf("[INFO] Starting server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(storage LocationStorage, template *Template) func(http.ResponseWriter, *http.Request) {
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
		timezone, err := time.LoadLocation(aliceRequest.Meta.Timezone)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		currentTime := time.Now().In(timezone)
		fmt.Println(currentTime)

		phrase := aliceRequest.Request.Command

		location := storage.Get(userID)

		response := getResponseStub(aliceRequest.Session)

		if !location.Completed && !location.InProgress {
			location.InProgress = true
			storage.Save(userID, location)

			response.Response.Tts = "Привет! А в каком городе и на какой станции метро, если оно есть, вы живете?"
			response.Response.Text = "Привет! А в каком городе и на какой станции метро, если оно есть, вы живете?"
		} else if !location.Completed && location.InProgress {
			newLocation, err := GetUserLocation(phrase)
			if err != nil {
				response.Response.Tts = "Что-то я не знаю такого адреса. А повторите пожалуйста в таком виде: \"Москва, метро Октябрьская\" или просто скажите название города, если метро нет, например \"Абакан\""
				response.Response.Text = "Что-то я не знаю такого адреса. А повторите пожалуйста в таком виде: \"Москва, метро Октябрьская\" или просто скажите название города, есди метро нет, например \"Абакан\""
			} else {
				newLocation.Completed = true
				newLocation.InProgress = false
				storage.Save(userID, newLocation)

				response.Response.Tts = "Отлично! На какой фильм вы хотите найти ближайшие сеансы?"
				response.Response.Text = "Отлично! На какой фильм вы хотите найти ближайшие сеансы?"
			}
		} else {
			extracted, ok := template.Matches(phrase)
			if !ok {
				response.Response.Tts = "Я вас почему то не понимаю, попробуйте еще. Например: \"Когда идут Звездные Войны\""
				response.Response.Text = "Я вас почему то не понимаю, попробуйте еще. Например: \"Когда идут Звездные Войны\""
			} else {
				movie, ok := extracted["movie"]
				if !ok {
					response.Response.Tts = "Я не поняла на какой фильм вы хотите, попробуйте еще"
					response.Response.Text = "Я не поняла на какой фильм вы хотите, попробуйте еще"
				} else {
					showtimes, err := GetRamblerShowtimes(movie, location.City, location.Subway)
					if err != nil {
						response.Response.Tts = "Я не поняла на какой фильм вы хотите, попробуйте еще"
						response.Response.Text = "Я не поняла на какой фильм вы хотите, попробуйте еще"
					}
				}
			}
		}

		json.NewEncoder(w).Encode(response)
	}
}
