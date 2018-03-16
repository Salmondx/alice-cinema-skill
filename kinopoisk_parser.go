package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/anaskhan96/soup"
	"golang.org/x/net/html/charset"
)

const SEARCH_URL_TEMPLATE = "https://www.kinopoisk.ru/index.php?kp_query=%s"
const SHOWTIME_URL_TEMPLATE = "https://kinopoisk.ru%s?search=%s"

// GetShowtimes returns a search result from kinopoisk.ru based on movie name and a user location
func GetShowtimes(movieName, city, region string) (*SearchResult, error) {
	name, link, err := findMovieInfo(movieName)
	if err != nil {
		return nil, err
	}
	// find a movie schedule
	showtimeRedirectLink := fmt.Sprintf(SHOWTIME_URL_TEMPLATE, link, url.QueryEscape(region))
	cinemas, err := findSchedule(showtimeRedirectLink)
	if err != nil {
		return nil, err
	}

	return &SearchResult{
		Movie:   name,
		Cinemas: cinemas,
	}, nil
}

func findMovieInfo(movieName string) (string, string, error) {
	resp, err := GetWithProxy(fmt.Sprintf(SEARCH_URL_TEMPLATE, url.QueryEscape(movieName)))
	if err != nil {
		return "", "", err
	}

	decodedHTML, err := decodeWindows(resp)
	if err != nil {
		return "", "", err
	}

	movieRoot := soup.HTMLParse(decodedHTML)
	searchResults := movieRoot.Find("div", "class", "search_results")
	if searchResults.Error != nil {
		return "", "", fmt.Errorf("failed to find a search results block")
	}
	// find matching movie
	topResult := searchResults.Find("div", "class", "element most_wanted")
	if topResult.Error != nil {
		return "", "", fmt.Errorf("movie with such name does not exists: %s", movieName)
	}

	// change to timezone based on region/city
	currentTime := time.Now()

	// find movie information
	infoBlock := topResult.Find("div", "class", "info").Find("p", "class", "name")
	name := infoBlock.Find("a").Text()
	year := infoBlock.Find("span", "class", "year").Text()

	if isNotOutdated(year, currentTime.Year()) {
		return "", "", fmt.Errorf("movie %s is too old and no possible showtimes will be found: %s", name, year)
	}

	// find link to the schedule
	linksBlock := topResult.Find("div", "class", "right").Find("ul", "class", "links")

	var link string
	for _, elem := range linksBlock.FindAll("li") {
		showtimeLink := elem.Find("a")
		if showtimeLink.Error != nil {
			continue
		}
		if showtimeLink.Text() == "сеансы" {
			attributes := showtimeLink.Attrs()
			if href, ok := attributes["href"]; ok {
				link = href
				break
			}
		}
	}
	if link == "" {
		return "", "", fmt.Errorf("failed to find a link to the schedule for movie: %s", name)
	}

	return name, link, nil
}

func findSchedule(redirectLink string) ([]Cinema, error) {
	showtimeRaw, err := GetWithProxy(redirectLink)
	if err != nil {
		return nil, err
	}

	showtimeHTML, err := decodeWindows(showtimeRaw)
	if err != nil {
		return nil, err
	}

	rootEl := soup.HTMLParse(showtimeHTML)
	scheduleItems := rootEl.Find("div", "class", "film-seances-page__seances")
	if scheduleItems.Error != nil {
		banElement := rootEl.Find("div", "class", "form form_state_image form_error_no form_help_yes form_audio_yes i-bem")
		if banElement.Error == nil {
			return nil, fmt.Errorf("ip banned, use another socks5 proxy")
		}
		return nil, fmt.Errorf("failed to find a schedule block")
	}

	cinemas := make([]Cinema, 0)
	for _, scheduleItem := range scheduleItems.FindAll("div", "class", "schedule-item") {
		cinemaInfoBlock := scheduleItem.Find("div", "class", "schedule-item__left")
		if cinemaInfoBlock.Error != nil {
			continue
		}
		cinemaName := cinemaInfoBlock.Find("a", "class", "schedule-cinema__name").Text()
		cinemaAddress := cinemaInfoBlock.Find("div", "class", "schedule-cinema__address").Text()
		cinemaSubway := cinemaInfoBlock.Find("div", "class", "schedule-cinema__metro").Text()

		showtimesBlock := scheduleItem.Find("div", "class", "schedule-item__right")
		if showtimesBlock.Error != nil {
			continue
		}

		showtimes := make([]Showtime, 0)
		for _, formatsRow := range showtimesBlock.FindAll("div", "class", "schedule-item__formats-row") {
			format := formatsRow.Find("span", "class", "schedule-item__formats-format").Text()

			for _, scheduleItem := range formatsRow.FindAll("span", "class", "schedule-item__session-button-wrapper") {
				rawTime := scheduleItem.Find("span", "class", "schedule-item__session-button schedule-item__session-button_active js-yaticket-button").Text()
				price := scheduleItem.Find("span", "class", "schedule-item__price").Text()
				if time, err := time.Parse("15:04", rawTime); err == nil {
					showtimes = append(showtimes, Showtime{
						Time:   time,
						Price:  price,
						Format: format,
					})
				}
			}

		}
		cinemas = append(cinemas, Cinema{
			Name:      cinemaName,
			Address:   cinemaAddress,
			Subway:    cinemaSubway,
			Showtimes: showtimes,
		})
	}
	return cinemas, nil
}

func GetWithProxy(siteURL string) (string, error) {
	requestURL := "https://api.proxycrawl.com/?token=&url=" + siteURL

	httpClient := &http.Client{}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.186 Safari/537.36")

	fmt.Println(siteURL)
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func isNotOutdated(year string, currentYear int) bool {
	if intYear, err := strconv.Atoi(year); err != nil {
		return intYear == currentYear || intYear-1 == currentYear
	}
	return false
}

func decodeWindows(htmlResp string) (string, error) {
	encodedResp, err := charset.NewReader(strings.NewReader(htmlResp), "CP1251")
	if err != nil {
		return "", err
	}
	validResp, err := ioutil.ReadAll(encodedResp)
	if err != nil {
		return "", err
	}

	return string(validResp), nil
}
