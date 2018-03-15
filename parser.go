package main

import "time"

type Showtime struct {
	Time   time.Time
	Price  string
	Format string
}

type Cinema struct {
	Name      string
	Address   string
	Subway    string
	Showtimes []Showtime
}

type SearchResult struct {
	Movie   string
	Cinemas []Cinema
}

type ShowtimeParser interface {
	GetShowtimes(movieName, city, region string) (*SearchResult, error)
}

type NoSuchMovieError struct {
	msg string
}

func (err NoSuchMovieError) Error() string {
	return "no such movie found: " + err.msg
}
