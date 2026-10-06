package openaicompatible

import "net/http"

type Attribution struct {
	ReferrerURL string
	Title       string
}

func (attribution Attribution) apply(httpRequest *http.Request) {
	if attribution.ReferrerURL != "" {
		httpRequest.Header.Set("HTTP-Referer", attribution.ReferrerURL)
	}
	if attribution.Title != "" {
		httpRequest.Header.Set("X-Title", attribution.Title)
	}
}
