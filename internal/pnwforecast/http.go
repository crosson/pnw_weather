package pnwforecast

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type SafeHTTP struct {
	Client    HTTPClient
	UserAgent string
}

func NewSafeHTTP() *SafeHTTP {
	return &SafeHTTP{
		Client:    &http.Client{Timeout: 12 * time.Second},
		UserAgent: "pnw-forecast-skill/0.1",
	}
}

func (h *SafeHTTP) GetJSON(endpoint string, out any) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return NewSkillError("UpstreamUnavailable", "invalid URL")
	}
	if _, ok := AllowedDomains[u.Hostname()]; !ok {
		return NewSkillError("UpstreamUnavailable", fmt.Sprintf("Disallowed domain: %s", u.Hostname()))
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", h.UserAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := h.Client.Do(req)
	if err != nil {
		return NewSkillError("UpstreamUnavailable", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return NewSkillError("RateLimited", fmt.Sprintf("rate limited by %s", u.Hostname()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return NewSkillError("UpstreamUnavailable", fmt.Sprintf("HTTP %d from %s", resp.StatusCode, u.Hostname()))
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return NewSkillError("ParseError", fmt.Sprintf("Invalid JSON from %s", u.Hostname()))
	}
	return nil
}
