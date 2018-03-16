package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fiam/gounidecode/unidecode"

	"github.com/anaskhan96/soup"
)

const ramblerSearchTemplate = "https://kassa.rambler.ru/search?search_str=%s"
const mskName = "москва"
const spbName = "санкт-петербург"
const nnName = "нижний новгород"

// NoSuchMovie fired when movie with given name is not found
var NoSuchMovie = NoSuchMovieError{}

var httpClient = &http.Client{}

// RamblerSearch contains info about matching movies
type RamblerSearch struct {
	Items []struct {
		Link string
		Name string
	}
}

// GetRamblerShowtimes retrieves showtimes info about the movie in provided region with sort logic based on user time
func GetRamblerShowtimes(movieName, city, region string, timezone *time.Location) (*SearchResult, error) {
	searchRes, err := getMovieDesciptions(movieName)
	if err != nil {
		return nil, err
	}
	if len(searchRes.Items) == 0 {
		return nil, NoSuchMovie
	}
	name := searchRes.Items[0].Name
	link := formatLink(searchRes.Items[0].Link, city)
	cinemas, err := getMovieShowtimes(link, city, region, timezone)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Movie:   name,
		Cinemas: cinemas,
	}, nil
}

func formatLink(link, city string) string {
	cityCode := ""
	city = strings.ToLower(city)
	switch city {
	case mskName:
		cityCode = "msk"
	case spbName:
		cityCode = "spb"
	case nnName:
		cityCode = "nnovgorod"
	default:
		cityCode = strings.Replace(unidecode.Unidecode(city), " ", "-", -1)
	}
	return strings.Replace(link, "movie/", cityCode+"/movie/", 1)
}

func getMovieDesciptions(movieName string) (*RamblerSearch, error) {
	resp, err := http.Get(fmt.Sprintf(ramblerSearchTemplate, url.QueryEscape(movieName)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var searchResult RamblerSearch
	err = json.NewDecoder(resp.Body).Decode(&searchResult)
	if err != nil {
		return nil, err
	}
	return &searchResult, nil
}

func getMovieShowtimes(link, city, region string, timezone *time.Location) ([]Cinema, error) {
	raw, err := soup.Get(link)
	if err != nil {
		return nil, err
	}
	root := soup.HTMLParse(raw)

	cinemas := make([]Cinema, 0)

	for _, item := range root.FindAll("div", "class", "rasp_item_in") {
		cinemaInfoBlock := item.Find("div", "class", "rasp_name")
		if cinemaInfoBlock.Error != nil {
			continue
		}
		cinemaName := cinemaInfoBlock.Find("div", "class", "rasp_title").Find("span", "class", "s-name").Text()
		addressBlock := cinemaInfoBlock.Find("div", "class", "rasp_place")
		if addressBlock.Error != nil {
			continue
		}
		address := addressBlock.Find("span").Text()
		subwayBlock := addressBlock.Find("div", "class", "rasp_place_metro")
		var subway string
		if subwayBlock.Error == nil {
			subway = subwayBlock.Text()
		}

		if region != "" {
			cityLower := strings.ToLower(city)

			// if user is from moscow of saint-petersburg, than follow a subway comparrison block
			if cityLower == mskName || cityLower == spbName {
				// if region is provided but there is no subway info for the cinema - skip it
				if subway == "" {
					continue
				}
				replacer := strings.NewReplacer("ё", "е",
					"Ё", "Е")
				if !strings.Contains(strings.ToLower(replacer.Replace(subway)), strings.ToLower(replacer.Replace(region))) {
					continue
				}
			}
		}

		scheduleBlock := item.Find("div", "class", "rasp_list")
		if scheduleBlock.Error != nil {
			continue
		}
		showtimes := make([]Showtime, 0)
		for _, showtimeBlock := range scheduleBlock.FindAll("li", "class", "btn_rasp") {
			attrs := showtimeBlock.Attrs()["class"]
			if strings.Contains(attrs, "inactive") {
				// should skip showtimes that are in past
				continue
			}

			if time, err := time.ParseInLocation("15:04", strings.TrimSpace(showtimeBlock.Text()), timezone); err == nil {
				if time.Hour() == 0 || time.Hour() == 1 {
					time = time.AddDate(0, 0, 1)
				}
				showtimes = append(showtimes, Showtime{Time: time})
			}
		}
		cinemas = append(cinemas, Cinema{
			Name:      cinemaName,
			Address:   address,
			Subway:    subway,
			Showtimes: showtimes,
		})
	}
	return cinemas, nil
}
