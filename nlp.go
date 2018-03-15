package main

import (
	"log"
	"math/rand"
	"regexp"
	"sort"
	"strings"
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

const CHANGE_ADDRESS = "Сменить адрес"
const GET_ADDRESS = "Мой адрес"

// MessageProcessor processes user phrases from Alice skill
type MessageProcessor struct {
	storage  LocationStorage
	template *Template
	answers  map[string][]string
}

// NewProcessor creates a new MessageProcessor with default templates
func NewProcessor(storage LocationStorage) *MessageProcessor {
	return &MessageProcessor{storage, Default(), availableAnswers()}
}

// Process processes through state machine logic an retrieves intents from user's phrases
func (p *MessageProcessor) Process(aliceRequest *AliceRequest) *AliceResponse {
	userID := aliceRequest.Session.UserID
	timezone, _ := time.LoadLocation(aliceRequest.Meta.Timezone)

	currentTime := time.Now().In(timezone)

	location := p.storage.Get(userID)

	session := aliceRequest.Session

	phrase := aliceRequest.Request.Command

	log.Printf("User %s says: %s", userID, phrase)

	// if location is unknown, we have to retrieve it from user
	if !location.Completed && !location.InProgress {
		location.InProgress = true
		p.storage.Save(userID, location)
		return say(session, p.getAnswer("ASK_LOCATION"))

	} else if !location.Completed && location.InProgress {
		// if location retrieval is in progress, we should complete it
		newLocation, err := GetUserLocation(phrase)
		if err != nil {
			return say(session, p.getAnswer("UNKNOWN_LOCATION"))
		}

		newLocation.Completed = true
		newLocation.InProgress = false
		p.storage.Save(userID, newLocation)

		return say(session, p.getAnswer("LOCATION_CONFIRMED"))
	} else {
		// buttons actions
		if phrase == GET_ADDRESS {
			log.Printf("User %s GET_ADDRESS request", userID)
			address := "Ваш адрес: город " + location.City
			if location.Subway != "" {
				address += ", метро " + location.Subway
			}
			return sayWithButtons(session, address)

		} else if phrase == CHANGE_ADDRESS {
			log.Printf("User %s CHANGE_ADDRESS request", userID)
			location.InProgress = true
			location.Completed = false
			p.storage.Save(userID, location)

			return say(session, p.getAnswer("CHANGE_ADDRESS"))
		}

		// if location exists, we should process requests as is
		extracted, ok := p.template.Matches(strings.ToLower(phrase))
		if !ok {
			return sayWithButtons(session, p.getAnswer("UNKNOWN_MOVIE"))
		}
		movie, ok := extracted["movie"]
		if !ok {
			return sayWithButtons(session, p.getAnswer("UNKNOWN_MOVIE"))
		}

		searchResult, err := GetRamblerShowtimes(movie, location.City, location.Subway, timezone)

		if err != nil {
			return sayWithButtons(session, p.getAnswer("UNKNOWN_MOVIE"))
		}
		log.Printf("User %s found cinemas with movie %s: %d", userID, movie, len(searchResult.Cinemas))
		if isNoShowtimes(searchResult) {
			return sayWithButtons(session, p.getAnswer("NO_SHOWTIMES"))
		}
		return sayWithButtons(session, constructShowtimesPhrase(searchResult, currentTime))
	}
}

func constructShowtimesPhrase(searchResult *SearchResult, userTime time.Time) string {
	showtimes := findNearestShowtimes(searchResult, userTime)
	var phrase string
	if len(showtimes) > 3 {
		// lots of cinemas nearby case
		phrase = "Я выбрала 3 кинотеатра с ближайшими сеансами. "
	}
	var builder strings.Builder

	for i := 0; i < len(showtimes); i++ {
		if i > 2 {
			break
		}

		showtime := showtimes[i]
		builder.WriteString("В " + showtime.Name + " ")
		if i == 0 {
			if len(showtime.Showtimes) == 1 {
				builder.WriteString("фильм начинается в " + showtime.Showtimes[0].Time.Format("15:04"))
			} else {
				builder.WriteString("сеансы начинаются в " + showtime.Showtimes[0].Time.Format("15:04"))
				builder.WriteString(" и в " + showtime.Showtimes[1].Time.Format("15:04") + " часов")
			}
			builder.WriteString(". ")
		} else {
			if len(showtime.Showtimes) == 1 {
				builder.WriteString("в " + showtime.Showtimes[0].Time.Format("15:04"))
			} else {
				builder.WriteString("в " + showtime.Showtimes[0].Time.Format("15:04"))
				builder.WriteString(" и в " + showtime.Showtimes[1].Time.Format("15:04") + " часов")
			}
			builder.WriteString(". ")
		}
	}

	return phrase + builder.String()
}

// returns top 2 nearest showtimes based on current time for each cinema
func findNearestShowtimes(searchResult *SearchResult, userTime time.Time) []Cinema {
	showtimes := make([]Cinema, 0)

	for _, cinema := range searchResult.Cinemas {
		sortedShowtimes := make([]Showtime, 0)

		for _, showtime := range cinema.Showtimes {
			if showtime.Time.After(userTime) {
				continue
			}
			sortedShowtimes = append(sortedShowtimes, showtime)
		}
		if len(sortedShowtimes) == 0 {
			continue
		}

		sort.Slice(sortedShowtimes, func(i, j int) bool { return sortedShowtimes[i].Time.Before(sortedShowtimes[j].Time) })
		copyCinema := Cinema{
			Name:      cinema.Name,
			Address:   cinema.Address,
			Subway:    cinema.Subway,
			Showtimes: sortedShowtimes,
		}
		showtimes = append(showtimes, copyCinema)
	}

	sort.Slice(showtimes, func(i, j int) bool {
		return showtimes[i].Showtimes[0].Time.Before(showtimes[j].Showtimes[0].Time)
	})
	return showtimes
}

func isNoShowtimes(search *SearchResult) bool {
	if search == nil {
		return true
	}
	for _, cinema := range search.Cinemas {
		if len(cinema.Showtimes) != 0 {
			return false
		}
	}
	return true
}

func sayWithButtons(session Session, phrase string) *AliceResponse {
	response := say(session, phrase)
	response.Response.Buttons = []Button{
		Button{
			Title: "Мой адрес",
			Hide:  true,
		},
		Button{
			Title: "Сменить адрес",
			Hide:  true,
		},
	}
	return response

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

func (p *MessageProcessor) getAnswer(tag string) string {
	answers := p.answers[tag]
	return answers[rand.Intn(len(answers))]
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
	answers["LOCATION_CONFIRMED"] = []string{
		"Отлично! На какой фильм вы хотите найти ближайшие сеансы?",
		"Хорошо, я запомнила. На какой фильм вы хотите сходить?",
	}
	answers["UNKNOWN_MOVIE"] = []string{
		"Я вас почему то не понимаю, скажите название фильма, например: \"Звездные Войны\"",
		"Почему-то не могу найти такой фильм, попробуйте сказать название фильма: \"Интерстеллар\"",
		"Что-то я не знаю такого фильма, скажите, пожалуйста, название фильма, например: \"Тор\"",
	}
	answers["NO_SHOWTIMES"] = []string{
		"Похоже на то, что в вашем регионе сейчас нет сеансов этого фильма",
		"Не могу найти сеансов. Похоже, что в вашем регионе сейчас этот фильм не идет",
		"По вашему адресу сейчас нет сеансов. Увы. Но вы всегда можете пойти на пробежку, спорт это очень полезно!",
		"Сеансов на сегодня я не вижу. Придется заняться чем-то ещё",
	}
	answers["CHANGE_ADDRESS"] = []string{
		"Хорошо, давайте поменяем адрес. Скажите в каком городе и на какой станции метро, если оно есть, вы живете",
	}
	return answers
}
