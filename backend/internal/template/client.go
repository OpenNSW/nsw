package template

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOneTradeBaseURL = "https://raw.githubusercontent.com/OpenNSW/one-trade-templates/main"

type oneTradeClient struct {
	baseURL    string
	httpClient *http.Client
}

type oneTradeManifest struct {
	ByID map[string]string `json:"byId"`
}

func newOneTradeClient(baseURL string) *oneTradeClient {
	return &oneTradeClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{},
	}
}

func (c *oneTradeClient) fetchManifest() (*oneTradeManifest, error) {
	data, err := c.fetchFile("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	var m oneTradeManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}
	return &m, nil
}

func (c *oneTradeClient) fetchFile(path string) ([]byte, error) {
	url := c.baseURL + "/" + path
	resp, err := c.httpClient.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
