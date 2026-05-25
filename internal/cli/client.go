package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
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
	return NewClientForServer(serverURL)
}

func NewClientForServer(serverURL string) *Client {
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	return &Client{
		HTTPClient: &http.Client{},
		ServerURL:  strings.TrimRight(serverURL, "/"),
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
	if len(m.Manifest) > 0 {
		m.Raw = m.Manifest
	} else {
		m.Raw = body
	}
	return &m, nil
}

func (c *Client) GetRawArtifact(org, namespace, name, version, path string) ([]byte, error) {
	requestURL := c.ServerURL + "/raw/" + org + "/" + namespace + "/" + name + "/" + escapeArtifactPath(path) + "@" + version
	req, err := http.NewRequest("GET", requestURL, nil)
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

func (c *Client) GetCoreSkill(name string) ([]byte, error) {
	requestURL := c.ServerURL + "/api/v1/core/skills/" + url.PathEscape(name) + "/SKILL.md"
	req, err := http.NewRequest("GET", requestURL, nil)
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

func (c *Client) GetPackage(org, namespace, name string) (*PackageDetailResponse, error) {
	requestURL := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name
	req, err := http.NewRequest("GET", requestURL, nil)
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

	var result PackageDetailResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode package response: %w", err)
	}
	return &result, nil
}

func (c *Client) SearchPackages(params SearchPackagesParams) (*SearchPackagesResponse, error) {
	var queryParts []string
	if params.Query != "" {
		values := url.Values{}
		values.Set("q", params.Query)
		queryParts = append(queryParts, values.Encode())
	}
	if params.ArtifactType != "" {
		values := url.Values{}
		values.Set("artifactType", params.ArtifactType)
		queryParts = append(queryParts, values.Encode())
	}
	requestURL := c.ServerURL + "/api/v1/search/packages"
	if len(queryParts) > 0 {
		requestURL += "?" + strings.Join(queryParts, "&")
	}

	req, err := http.NewRequest("GET", requestURL, nil)
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

	var result SearchPackagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode search response: %w", err)
	}
	return &result, nil
}

func (c *Client) CreatePackage(reqBody CreatePackageRequest) (*PackageResponse, error) {
	var result PackageResponse
	err := c.doJSON("POST", c.ServerURL+"/api/v1/packages", reqBody, http.StatusCreated, &result)
	return &result, err
}

func (c *Client) GetVersion(org, namespace, name, version string) (*VersionResponse, error) {
	requestURL := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions/" + version
	req, err := http.NewRequest("GET", requestURL, nil)
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
	var result VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode version response: %w", err)
	}
	return &result, nil
}

func (c *Client) CreateDraft(org, namespace, name string, reqBody CreateDraftRequest) (*VersionResponse, error) {
	requestURL := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions"
	var result VersionResponse
	err := c.doJSON("POST", requestURL, reqBody, http.StatusCreated, &result)
	return &result, err
}

func (c *Client) UploadArtifact(org, namespace, name, version, path string, content []byte, contentType string) (*ArtifactResponse, error) {
	requestURL := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/artifacts/" + escapeArtifactPath(path)
	req, err := http.NewRequest("PUT", requestURL, bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
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
	var result ArtifactResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode upload response: %w", err)
	}
	return &result, nil
}

func (c *Client) PublishVersion(org, namespace, name, version string) (*VersionResponse, error) {
	requestURL := c.ServerURL + "/api/v1/packages/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/publish"
	var result VersionResponse
	err := c.doJSON("POST", requestURL, map[string]string{}, http.StatusOK, &result)
	return &result, err
}

func (c *Client) SubmitReview(org, namespace, name, version string) error {
	requestURL := c.ServerURL + "/api/v1/reviews/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/submit"
	return c.doJSON("POST", requestURL, map[string]string{}, http.StatusOK, nil)
}

func (c *Client) ApprovalStatus(org, namespace, name, version string) (*ApprovalStatusResponse, error) {
	requestURL := c.ServerURL + "/api/v1/reviews/" + org + "/" + namespace + "/" + name + "/versions/" + version + "/approval-status"
	var result ApprovalStatusResponse
	err := c.doJSON("GET", requestURL, nil, http.StatusOK, &result)
	return &result, err
}

func (c *Client) doJSON(method, requestURL string, body any, wantStatus int, out any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequest(method, requestURL, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.setAuth(req)
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != wantStatus {
		return httpError(resp)
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
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
			Code      string         `json:"code"`
			Message   string         `json:"message"`
			RequestID string         `json:"requestId"`
			Details   map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil && apiErr.Error.Message != "" {
		requestID := apiErr.Error.RequestID
		if requestID == "" {
			requestID = resp.Header.Get("X-Request-Id")
		}
		return &APIError{Status: resp.StatusCode, Code: apiErr.Error.Code, Message: apiErr.Error.Message, RequestID: requestID, Details: apiErr.Error.Details}
	}
	return fmt.Errorf("server error %d: %s", resp.StatusCode, string(body))
}

type APIError struct {
	Status    int
	Code      string
	Message   string
	RequestID string
	Details   map[string]any
}

func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("server error %d: %s (%s, request %s)", e.Status, e.Message, e.Code, e.RequestID)
	}
	return fmt.Sprintf("server error %d: %s (%s)", e.Status, e.Message, e.Code)
}

func escapeArtifactPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
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

type CreatePackageRequest struct {
	Org         string `json:"org"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type CreateDraftRequest struct {
	Version    string `json:"version"`
	Visibility string `json:"visibility"`
}

type PackageResponse struct {
	Org         string `json:"org"`
	Namespace   string `json:"namespace"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
	Lifecycle   string `json:"lifecycle"`
}

type PackageDetailResponse struct {
	Org           string           `json:"org"`
	Namespace     string           `json:"namespace"`
	Name          string           `json:"name"`
	DisplayName   string           `json:"displayName"`
	Description   string           `json:"description"`
	Visibility    string           `json:"visibility"`
	Lifecycle     string           `json:"lifecycle"`
	LatestVersion string           `json:"latestVersion"`
	StableVersion string           `json:"stableVersion"`
	Versions      []PackageVersion `json:"versions"`
}

type PackageVersion struct {
	Version     string `json:"version"`
	Digest      string `json:"digest"`
	Lifecycle   string `json:"lifecycle"`
	Channel     string `json:"channel"`
	PublishedAt string `json:"publishedAt"`
}

type VersionResponse struct {
	Org         string `json:"org"`
	Namespace   string `json:"namespace"`
	Package     string `json:"package"`
	Version     string `json:"version"`
	Lifecycle   string `json:"lifecycle"`
	Visibility  string `json:"visibility"`
	Digest      string `json:"digest"`
	PublishedAt string `json:"publishedAt"`
}

type ArtifactResponse struct {
	Path       string `json:"path"`
	Type       string `json:"type"`
	Digest     string `json:"digest"`
	SizeBytes  int64  `json:"sizeBytes"`
	TargetPath string `json:"targetPath"`
}

type SearchPackagesParams struct {
	Query        string
	ArtifactType string
}

type SearchPackagesResponse struct {
	Items      []PackageSummary `json:"items"`
	NextCursor *string          `json:"nextCursor"`
}

type PackageSummary struct {
	Org           string `json:"org"`
	Namespace     string `json:"namespace"`
	Name          string `json:"name"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description"`
	Visibility    string `json:"visibility"`
	Lifecycle     string `json:"lifecycle"`
	LatestVersion string `json:"latestVersion,omitempty"`
	StableVersion string `json:"stableVersion,omitempty"`
}

type ApprovalStatusResponse struct {
	HasEnoughApprovals bool  `json:"hasEnoughApprovals"`
	CurrentCount       int64 `json:"currentCount"`
	RequiredCount      int   `json:"requiredCount"`
}

type UpdateCheckRequest struct {
	Package             string `json:"package"`
	CurrentVersion      string `json:"currentVersion"`
	CurrentDigest       string `json:"currentDigest"`
	Channel             string `json:"channel"`
	StrictCompatibility bool   `json:"strictCompatibility"`
	Target              Target `json:"target"`
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
