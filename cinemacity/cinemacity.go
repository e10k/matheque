// Package cinemacity offers functionality for fetching films from a JSON resource
// and for fetching the romanian names by scraping movies' HTML pages.
package cinemacity

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
)

type Film struct {
	Id         string `json:"code"`
	Name       string `json:"featureTitle"`
	Link       string `json:"url"`
	PosterLink string `json:"posterSrc"`
}

type Event struct {
	Id            string `json:"id"`
	FilmId        string `json:"filmId"`
	CinemaId      string `json:"cinemaId"`
	BusinessDay   string `json:"businessDay"`
	EventDateTime string `json:"eventDateTime"`
	BookingLink   string `json:"bookingLink"`
	Auditorium    string `json:"auditorium"`
}

type Body struct {
	Films []Film `json:"posters"`
}

type FilmsList struct {
	Body `json:"body"`
}

func GetFilms() ([]Film, error) {
	url := "https://www.cinemacity.ro/ro/data-api-service/v1/feed/10107/byName/now-playing?lang=en_GB"

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var films FilmsList
	if err := json.Unmarshal(body, &films); err != nil {
		return nil, err
	}

	var filmsToReturn []Film

	for _, film := range films.Films {
		filmsToReturn = append(filmsToReturn, film)
	}

	return filmsToReturn, nil
}

func GetRomanianName(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`var featureName = "(?P<Title>[^"]+)"`)
	matches := re.FindStringSubmatch(string(body))

	if len(matches) != 2 {
		return "", errors.New("couldn't scrape the film's romanian name")
	}

	return matches[1], err
}
