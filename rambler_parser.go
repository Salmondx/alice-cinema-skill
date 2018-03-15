package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
)

const RAMBLER_SEARCH_TEMPLATE = "https://kassa.rambler.ru/search?search_str=%s"

type RamblerSearch struct {
	Items []struct {
		Link string
		Name string
	}
}

func GetRamblerShowtimes(movieName, city, region string) (*SearchResult, error) {
	searchRes, err := getMovieDesciptions(movieName)
	if err != nil {
		return nil, err
	}
	if len(searchRes.Items) == 0 {
		return nil, NoSuchMovieError{movieName}
	}
	name := searchRes.Items[0].Name
	cinemas, err := getMovieShowtimes(searchRes.Items[0].Link, region)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Movie:   name,
		Cinemas: cinemas,
	}, nil
}

func getMovieDesciptions(movieName string) (*RamblerSearch, error) {
	resp, err := http.Get(fmt.Sprintf(RAMBLER_SEARCH_TEMPLATE, url.QueryEscape(movieName)))
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

func getMovieShowtimes(link, region string) ([]Cinema, error) {
	raw, err := soup.Get(link)
	if err != nil {
		return nil, err
	}
	root := soup.HTMLParse(raw)

	cinemas := make([]Cinema, 0)

	for _, item := range root.FindAll("div", "class", "rasp_item_in") {
		cinemaInfoBlock := item.Find("div", "class", "rasp_name")
		cinemaName := cinemaInfoBlock.Find("div", "class", "rasp_title").Find("span", "class", "s-name").Text()
		addressBlock := cinemaInfoBlock.Find("div", "class", "rasp_place s-place")
		address := addressBlock.Find("span").Text()
		subwayBlock := addressBlock.Find("div", "class", "rasp_place_metro")
		var subway string
		if subwayBlock.Error == nil {
			subway = subwayBlock.Text()
		}

		if region != "" {
			// if region is provided but there is no subway info for the cinema - skip it
			if subway == "" {
				continue
			}
			if !strings.Contains(strings.ToLower(subway), strings.ToLower(region)) {
				continue
			}
		}

		scheduleBlock := item.Find("div", "class", "rasp_list")
		if scheduleBlock.Error != nil {
			continue
		}
		showtimes := make([]Showtime, 0)
		for _, showtimeBlock := range scheduleBlock.FindAll("li", "class", "btn_rasp") {
			if time, err := time.Parse("15:04", strings.TrimSpace(showtimeBlock.Text())); err == nil {
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
