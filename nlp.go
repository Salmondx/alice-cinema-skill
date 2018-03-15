package main

import (
	"fmt"
	"regexp"
	"time"
)

type Template struct {
	regexps []*regexp.Regexp
}

func New(regExpressions ...string) (*Template, error) {
	regexps := make([]*regexp.Regexp, 0)
	for _, phrase := range regExpressions {
		re, err := regexp.Compile(phrase)
		if err != nil {
			return nil, err
		}

		regexps = append(regexps, re)
	}
	return &Template{regexps}, nil
}

func (t *Template) Matches(phrase string) (map[string]string, bool) {
	for _, regexp := range t.regexps {
		match := regexp.FindStringSubmatch(phrase)
		if len(match) == 0 {
			continue
		}
		result := make(map[string]string)
		for i, name := range regexp.SubexpNames() {
			if i != 0 && name != "" {
				result[name] = match[i]
			}
		}
		return result, true
	}
	return nil, false
}

func Default() *Template {
	t, _ := New(
		`.*(?:сеансы|расписание|время|когда идет|когда идут|когда будет|когда показывают).*(?:фильма|фильм|кино|кинофильма|кинофильм) (?P<movie>.*)`,
		`.*(?:сеансы|расписание|время|когда идет|когда идут|когда будет|когда показывают|хочу|хотелось).*(?:фильма|фильм|кино|кинофильма|кинофильм)(?: на | для | у | +)(?P<movie>.*)`,
		`.*(?:сеансы|расписание|время|когда идет|когда идут|когда будет|когда показывают)(?: на | для | у | +)(?P<movie>.*)`,
		`.*(?:когда|в какое|когда будет|во сколько|время|время начала) .* (?:фильма|фильм|кино|кинофильма|кинофильм) (?P<movie>.*)`,
		`.*(?:когда|в какое|когда будет|во сколько|во сколько будет|когда идет|время начала|давай|хочу) .* (?:на|в) (?P<movie>.*)`,
		`.*(?:фильма|фильм|кино|кинофильма|кинофильм) (?P<movie>.*)`,
		`.*(?:хочу посмотреть|глянуть|хочу|хотелось) .* (?:фильма|фильм|кино|кинофильма|кинофильм) (?P<movie>.*)`,
		`.*(?:хочу посмотреть|глянуть|хочу|хотелось)(?: на | для | у | +)(?P<movie>.*)`,
		`.*(?:хочу посмотреть|глянуть) (?P<movie>.*)`,
		`.*(?:когда|когда будет|в какое|во сколько|во сколько|когда идет|когда идут) (?P<movie>.*)`,
		`.*(?:сеансы|расписание|время|когда идет|когда идут|когда будет|когда показывают) (?P<movie>.*)`,
		`(?:на|в|хочу|давай) (?P<movie>.*)`,
		`(?P<movie>.*)`,
	)
	return t
}

// MessageProcessor processes user phrases from Alice skill
type MessageProcessor struct {
	storage  LocationStorage
	template *Template
}

// NewProcessor creates a new MessageProcessor with default templates
func NewProcessor(storage LocationStorage) *MessageProcessor {
	return &MessageProcessor{storage, Default()}
}

// Process processes through state machine logic an retrieves intents from user's phrases
func (p *MessageProcessor) Process(aliceRequest *AliceRequest) *AliceResponse {
	userID := aliceRequest.Session.UserID
	timezone, _ := time.LoadLocation(aliceRequest.Meta.Timezone)

	currentTime := time.Now().In(timezone)
	fmt.Println(currentTime)

	location := p.storage.Get(userID)

	session := aliceRequest.Session

	phrase := aliceRequest.Request.Command

	// if location is unknown, we have to retrieve it from user
	if !location.Completed && !location.InProgress {
		location.InProgress = true
		p.storage.Save(userID, location)
		return say(session, "Привет! А в каком городе и на какой станции метро, если оно есть, вы живете?")

	} else if !location.Completed && location.InProgress {
		// if location retrieval is in progress, we should complete it
		newLocation, err := GetUserLocation(phrase)
		if err != nil {
			return say(session, "Что-то я не знаю такого адреса. А повторите пожалуйста в таком виде: \"Москва, метро Октябрьская\" или просто скажите название города, если метро нет, например \"Абакан\"")
		}

		newLocation.Completed = true
		newLocation.InProgress = false
		p.storage.Save(userID, newLocation)

		return say(session, "Отлично! На какой фильм вы хотите найти ближайшие сеансы?")
	} else {
		// if location exists, we should process requests as is
		extracted, ok := p.template.Matches(phrase)
		if !ok {
			return say(session, "Я вас почему то не понимаю, попробуйте еще. Например: \"Когда идут Звездные Войны\"")
		} else {
			movie, ok := extracted["movie"]
			if !ok {
				return say(session, "Я не поняла на какой фильм вы хотите, попробуйте еще")
			} else {
				showtimes, err := GetRamblerShowtimes(movie, location.City, location.Subway)
				if err != nil {
					return say(session, "Я не поняла на какой фильм вы хотите, попробуйте еще")
				}
			}
		}
	}
	return nil
}

func say(session Session, phrase string) *AliceResponse {
	response := getResponseStub(session)
	response.Response.Tts = phrase
	response.Response.Text = phrase
	return response
}

func sayTerminal(session Session, phrase string) *AliceResponse {
	response := say(session, phrase)
	response.Response.EndSession = true
	return response
}

func getResponseStub(session Session) *AliceResponse {
	return &AliceResponse{
		Version: "1.0",
		Session: session,
	}
}

func availableAnswers() map[string][]string {
	answers := make(map[string][]string)

	answers["ASK_LOCATION"] = []string{
		"Привет! А в каком городе и на какой станции метро, если оно есть, вы живете?",
	}
	answers["UNKNOWN_LOCATION"] = []string{
		"Что-то я не знаю такого адреса. А повторите пожалуйста в таком виде: \"Москва, метро Октябрьская\" или просто скажите название города, если метро нет, например \"Абакан\"",
		"Этот адрес мне неизвестен, повторите ещё",
		"Такого адреса я не знаю, повторите в виде \"Москва, метро Дмитровская\" или скажите просто название города, если у вас нет метро\"",
		"Не могу найти такой адрес, попробуйте ещё",
	}
	answers[]
}
