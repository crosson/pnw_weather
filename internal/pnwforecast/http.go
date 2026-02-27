package pnwforecast

import (
	"encoding/json"
	"fmt"
	"io"
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
	raw, err := h.getRaw(endpoint, "application/json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(raw, out); err != nil {
		u, _ := url.Parse(endpoint)
		host := ""
		if u != nil {
			host = u.Hostname()
		}
		return NewSkillError("ParseError", fmt.Sprintf("Invalid JSON from %s", host))
	}
	return nil
}

func (h *SafeHTTP) GetText(endpoint string) (string, error) {
	raw, err := h.getRaw(endpoint, "text/html,application/xhtml+xml")
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (h *SafeHTTP) getRaw(endpoint string, accept string) ([]byte, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, NewSkillError("UpstreamUnavailable", "invalid URL")
	}
	if _, ok := AllowedDomains[u.Hostname()]; !ok {
		return nil, NewSkillError("UpstreamUnavailable", fmt.Sprintf("Disallowed domain: %s", u.Hostname()))
	}

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", h.UserAgent)
	req.Header.Set("Accept", accept)

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, NewSkillError("UpstreamUnavailable", err.Error())
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, NewSkillError("RateLimited", fmt.Sprintf("rate limited by %s", u.Hostname()))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, NewSkillError("UpstreamUnavailable", fmt.Sprintf("HTTP %d from %s", resp.StatusCode, u.Hostname()))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewSkillError("UpstreamUnavailable", fmt.Sprintf("Read error from %s", u.Hostname()))
	}
	return raw, nil
}
