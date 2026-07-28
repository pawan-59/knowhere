// Package devtron is a client for the Devtron orchestrator API. It tracks
// application deployments/rollouts and the Devtron release/version running.
//
// Endpoints target the Devtron orchestrator (`/orchestrator/...`). Some paths
// vary by Devtron version; they are centralized here so they are easy to tune.
package devtron

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"knowhere/internal/config"
)

type Client struct {
	cfg  config.DevtronConfig
	http *http.Client
}

func New(cfg config.DevtronConfig) *Client {
	return &Client{
		cfg:  cfg,
		http: &http.Client{Timeout: 25 * time.Second},
	}
}

func (c *Client) Configured() bool {
	return c.cfg.BaseURL != "" && c.cfg.APIToken != ""
}

// get performs an authenticated GET and decodes the standard Devtron envelope
// `{ "code": .., "status": .., "result": <T> }` into out.
func (c *Client) get(ctx context.Context, path string, query url.Values, out any) error {
	u := strings.TrimRight(c.cfg.BaseURL, "/") + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("token", c.cfg.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("devtron get %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("devtron get %s: status %d", path, resp.StatusCode)
	}

	env := struct {
		Result json.RawMessage `json:"result"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return fmt.Errorf("devtron decode %s: %w", path, err)
	}
	if out == nil || len(env.Result) == 0 {
		return nil
	}
	return json.Unmarshal(env.Result, out)
}

// Version describes the Devtron release currently deployed.
type Version struct {
	Version    string `json:"version"`
	GitCommit  string `json:"gitCommit,omitempty"`
	ServerMode string `json:"serverMode,omitempty"`
}

// ServerVersion returns the Devtron server/release version.
func (c *Client) ServerVersion(ctx context.Context) (*Version, error) {
	var v Version
	if err := c.get(ctx, "/orchestrator/server/version", nil, &v); err != nil {
		return nil, err
	}
	return &v, nil
}

// App is an application known to Devtron.
type App struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ListApps returns the applications registered in Devtron.
func (c *Client) ListApps(ctx context.Context) ([]App, error) {
	var apps []App
	if err := c.get(ctx, "/orchestrator/app/autocomplete", nil, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

// Deployment is a single app/environment deployment status entry.
type Deployment struct {
	AppID           int       `json:"appId"`
	AppName         string    `json:"appName"`
	EnvironmentName string    `json:"environmentName"`
	Status          string    `json:"deploymentStatus"`
	DeployedBy      string    `json:"deployedBy,omitempty"`
	LastDeployed    time.Time `json:"lastDeployedTime"`
	Version         string    `json:"deployedVersion,omitempty"`
}

// Deployments returns deployment/rollout status across apps and environments.
// Uses the app-group / deployment status listing endpoint.
func (c *Client) Deployments(ctx context.Context) ([]Deployment, error) {
	var raw struct {
		DeploymentStatuses []Deployment `json:"deploymentStatuses"`
	}
	// Some Devtron versions return the list directly; try the wrapper first,
	// then fall back to a bare array.
	if err := c.get(ctx, "/orchestrator/app/deployment/status/list", nil, &raw); err == nil && len(raw.DeploymentStatuses) > 0 {
		return raw.DeploymentStatuses, nil
	}
	var list []Deployment
	if err := c.get(ctx, "/orchestrator/app/deployment/status/list", nil, &list); err != nil {
		return nil, err
	}
	return list, nil
}
