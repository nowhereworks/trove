package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const defaultServerURL = "http://localhost:8080"

type Client struct {
	HTTPClient *http.Client
	ServerURL  string
	Token      string
}

func NewClient() *Client {
	serverURL := os.Getenv("TROVE_SERVER_URL")
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	return &Client{
		HTTPClient: &http.Client{},
		ServerURL:  serverURL,
		Token:      os.Getenv("TROVE_TOKEN"),
	}
}

func (c *Client) Resolve(org, namespace, name, selector string) (*ResolveResponse, error) {
	url := c.ServerURL + "/api/v1/resolve/" + org + "/" + namespace + "/" + name + "@" + selector
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp)
	}

	var result ResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode resolve response: %w", err)
	}
	return &result, nil
}

func (c *Client) GetManifest(org, namespace, name, version string) (*ManifestResponse, error) {
	url := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/manifest"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read manifest body: %w", err)
	}

	var m ManifestResponse
	if err := json.Unmarshal(body, &m); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}
	m.Raw = body
	return &m, nil
}

func (c *Client) GetRawArtifact(org, namespace, name, version, path string) ([]byte, error) {
	url := c.ServerURL + "/raw/" + org + "/" + namespace + "/" + name + "/" + version + "/" + path
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) GetArchive(org, namespace, name, version, format string) ([]byte, error) {
	url := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/archive." + format
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp)
	}

	return io.ReadAll(resp.Body)
}

func (c *Client) CheckUpdate(req UpdateCheckRequest) (*UpdateCheckResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	url := c.ServerURL + "/api/v1/updates/check"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	c.setAuth(httpReq)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, httpError(resp)
	}

	var result UpdateCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode update check response: %w", err)
	}
	return &result, nil
}

func (c *Client) setAuth(req *http.Request) {
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
}

func httpError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	var apiErr struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		return fmt.Errorf("server error %d: %s (%s)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Code)
	}
	return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
}

type ResolveResponse struct {
	Org             string `json:"org"`
	Namespace       string `json:"namespace"`
	Package         string `json:"package"`
	Selector        string `json:"selector"`
	ResolvedVersion string `json:"resolvedVersion"`
	Digest          string `json:"digest"`
	ManifestURL     string `json:"manifestUrl"`
	ArchiveURL      string `json:"archiveUrl"`
}

type ManifestResponse struct {
	Org       string          `json:"org"`
	Namespace string          `json:"namespace"`
	Package   string          `json:"package"`
	Version   string          `json:"version"`
	Digest    string          `json:"digest"`
	Manifest  json.RawMessage `json:"manifest"`
	Raw       []byte
}

type UpdateCheckRequest struct {
	Package             string   `json:"package"`
	CurrentVersion      string   `json:"currentVersion"`
	CurrentDigest       string   `json:"currentDigest"`
	Channel             string   `json:"channel"`
	StrictCompatibility bool     `json:"strictCompatibility"`
	Target              Target   `json:"target"`
}

type Target struct {
	Tool          string `json:"tool"`
	ToolVersion   string `json:"toolVersion"`
	Runtime       string `json:"runtime"`
	ModelFamily   string `json:"modelFamily"`
	ContextWindow int    `json:"contextWindow"`
}

type UpdateCheckResponse struct {
	UpdateAvailable        bool   `json:"updateAvailable"`
	LatestVersion          string `json:"latestVersion"`
	LatestDigest           string `json:"latestDigest"`
	Compatibility          string `json:"compatibility"`
	RequiresManualApproval bool   `json:"requiresManualApproval"`
	ChangelogURL           string `json:"changelogUrl"`
}
