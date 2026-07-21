// Package zoho is a client for the Zoho Desk API (ticket monitoring).
package zoho

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"central-devtron/internal/config"
)

// Client talks to the Zoho Desk REST API using an OAuth refresh token.
// Access tokens are short-lived (~1h) so we cache and auto-refresh them.
type Client struct {
	cfg  config.ZohoConfig
	http *http.Client

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

func New(cfg config.ZohoConfig) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 20 * time.Second},
	}
}

// Configured reports whether the minimum credentials are present.
func (c *Client) Configured() bool {
	return c.cfg.ClientID != "" && c.cfg.ClientSecret != "" && c.cfg.RefreshToken != "" && c.cfg.OrgID != ""
}

// token returns a valid access token, refreshing it if expired.
func (c *Client) token(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}

	form := url.Values{}
	form.Set("refresh_token", c.cfg.RefreshToken)
	form.Set("client_id", c.cfg.ClientID)
	form.Set("client_secret", c.cfg.ClientSecret)
	form.Set("grant_type", "refresh_token")

	endpoint := strings.TrimRight(c.cfg.AccountsBase, "/") + "/oauth/v2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("zoho token request: %w", err)
	}
	defer resp.Body.Close()

	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("zoho token decode: %w", err)
	}
	if tr.Error != "" || tr.AccessToken == "" {
		return "", fmt.Errorf("zoho token error: %s", tr.Error)
	}

	c.accessToken = tr.AccessToken
	// Refresh a minute early to avoid edge-of-expiry failures.
	c.tokenExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return c.accessToken, nil
}

// get performs an authenticated GET against the Desk API and decodes into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	tok, err := c.token(ctx)
	if err != nil {
		return err
	}

	u := strings.TrimRight(c.cfg.APIBase, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Zoho-oauthtoken "+tok)
	req.Header.Set("orgId", c.cfg.OrgID)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("zoho get %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	if resp.StatusCode >= 300 {
		return fmt.Errorf("zoho get %s: status %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Ticket is a trimmed Zoho Desk ticket relevant to the dashboard.
type Ticket struct {
	ID           string    `json:"id"`
	Number       string    `json:"ticketNumber"`
	Subject      string    `json:"subject"`
	Status       string    `json:"status"`
	Priority     string    `json:"priority"`
	Channel      string    `json:"channel"`
	CreatedTime  time.Time `json:"createdTime"`
	ModifiedTime time.Time `json:"modifiedTime"`
	DueDate      *string   `json:"dueDate,omitempty"`
	Assignee     string    `json:"assignee,omitempty"`
	Contact      string    `json:"contact,omitempty"`
}

// ListTickets fetches recent tickets. limit is capped at 100 by Zoho.
func (c *Client) ListTickets(ctx context.Context, limit int, status string) ([]Ticket, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	q := url.Values{}
	q.Set("limit", strconv.Itoa(limit))
	q.Set("sortBy", "-modifiedTime")
	if status != "" {
		q.Set("status", status)
	}

	var raw struct {
		Data []struct {
			ID           string    `json:"id"`
			TicketNumber string    `json:"ticketNumber"`
			Subject      string    `json:"subject"`
			Status       string    `json:"status"`
			Priority     string    `json:"priority"`
			Channel      string    `json:"channel"`
			CreatedTime  time.Time `json:"createdTime"`
			ModifiedTime time.Time `json:"modifiedTime"`
			DueDate      *string   `json:"dueDate"`
			Assignee     struct {
				Name string `json:"firstName"`
			} `json:"assignee"`
			Contact struct {
				Name string `json:"firstName"`
			} `json:"contact"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v1/tickets", q, &raw); err != nil {
		return nil, err
	}

	tickets := make([]Ticket, 0, len(raw.Data))
	for _, d := range raw.Data {
		tickets = append(tickets, Ticket{
			ID: d.ID, Number: d.TicketNumber, Subject: d.Subject,
			Status: d.Status, Priority: d.Priority, Channel: d.Channel,
			CreatedTime: d.CreatedTime, ModifiedTime: d.ModifiedTime, DueDate: d.DueDate,
			Assignee: d.Assignee.Name, Contact: d.Contact.Name,
		})
	}
	return tickets, nil
}

// CountByStatus returns ticket counts grouped by status using Zoho's count API.
func (c *Client) CountByStatus(ctx context.Context) (map[string]int, error) {
	// The Desk count API accepts a field to group by.
	q := url.Values{}
	q.Set("field", "status")
	var raw struct {
		Data []struct {
			Value string `json:"val"`
			Count int    `json:"count"`
		} `json:"data"`
	}
	if err := c.get(ctx, "/api/v1/ticketsCountByField", q, &raw); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(raw.Data))
	for _, d := range raw.Data {
		out[d.Value] = d.Count
	}
	return out, nil
}
