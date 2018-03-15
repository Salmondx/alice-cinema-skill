package main

import (
	"regexp"
	"testing"
)

func TestTemplate(t *testing.T) {
	re, _ := regexp.Compile(`.* (?:movie|cinema) (?P<movie>.*)`)
	template := Template{[]*regexp.Regexp{
		re,
	}}

	extracted, matched := template.Matches("i want to go to the cinema black panther")
	if !matched {
		t.Error("failed to match a phrase")
	}
	extr, ok := extracted["movie"]
	if !ok {
		t.Error("failed to retrieve the extracted parameter")
	}
	if extr != "black panther" {
		t.Errorf("wrong term extracted: %s", extr)
	}
}

func TestTemplates(t *testing.T) {
	var td = []struct {
		Phrase    string
		Extracted string
	}{
		{"черная пантера", "черная пантера"},
		{"найди сеансы фильма пассажир", "пассажир"},
		{"хочу посмотреть черную пантеру", "черную пантеру"},
		{"когда будет фильм черная пантера", "черная пантера"},
		{"когда сходить на излом времени", "излом времени"},
		{"во сколько будет фильм пассажир", "пассажир"},
		{"во сколько пассажир", "пассажир"},
		{"какие сеансы у излома времени", "излома времени"},
		{"расписание черной пантеры", "черной пантеры"},
		{"время кино пассажир", "пассажир"},
		{"когда идет лара крофт", "лара крофт"},
		{"расписание пассажира", "пассажира"},
		{"хочу в кино на пассажира", "пассажира"},
		{"время начала фильма излом времени", "излом времени"},
	}

	template := Default()

	for _, tr := range td {
		extr, matched := template.Matches(tr.Phrase)
		if !matched {
			t.Fatalf("failed to match: %s", tr.Phrase)
		}
		val, ok := extr["movie"]
		if !ok {
			t.Fatalf("value no found for phrase %s: %s", tr.Phrase, tr.Extracted)
		}
		if val != tr.Extracted {
			t.Fatalf("extracted not matched %s: %s - %s", tr.Phrase, val, tr.Extracted)
		}
	}
}
