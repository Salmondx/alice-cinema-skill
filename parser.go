package main

import "time"

// Showtime containes info about movie seance
type Showtime struct {
	Time   time.Time
	Price  string
	Format string
}

// Cinema contains info about cinema and a slice of showtimes
type Cinema struct {
	Name      string
	Address   string
	Subway    string
	Showtimes []Showtime
}

// SearchResult contains info about movie seances
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
