// Package storage handles file storage operations including upload and download URL generation.
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/OpenNSW/nsw/oga/pkg/httpclient"
)

// Service handles storage operations (upload/download URLs)
type Service interface {
	// GetDownloadURL fetches a download URL for a key from the main backend.
	GetDownloadURL(ctx context.Context, key string) (map[string]any, error)

	// CreateUploadURL proxies an upload initialization request to the main backend.
	CreateUploadURL(ctx context.Context, payload []byte) (map[string]any, error)
}

type service struct {
	httpClient *httpclient.Client
}

// NewService creates a new storage service instance
func NewService(httpClient *httpclient.Client) Service {
	return &service{
		httpClient: httpClient,
	}
}

// GetDownloadURL returns a download URL for a file stored in the main backend.
// It calls the backend's metadata endpoint to retrieve a (possibly presigned) download URL.
func (s *service) GetDownloadURL(ctx context.Context, key string) (map[string]any, error) {
	apiURL := fmt.Sprintf("uploads/%s", url.PathEscape(key))
	resp, err := s.httpClient.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch upload metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.WarnContext(ctx, "failed to fetch upload metadata",
			"key", key, "status", resp.Status)
		return nil, fmt.Errorf("failed to fetch upload metadata, status code: %d", resp.StatusCode)
	}

	var metadata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode upload metadata: %w", err)
	}

	if metadata["download_url"] == nil || metadata["download_url"] == "" {
		return nil, fmt.Errorf("metadata response missing download_url")
	}

	slog.InfoContext(ctx, "resolved download URL from metadata", "key", key, "downloadURL", metadata["download_url"])
	return metadata, nil
}

// CreateUploadURL proxies an upload initialization request to the main backend.
func (s *service) CreateUploadURL(ctx context.Context, payload []byte) (map[string]any, error) {
	var req struct {
		Filename string `json:"filename"`
		MimeType string `json:"mime_type"`
		Size     int64  `json:"size"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return nil, fmt.Errorf("invalid upload request format: %w", err)
	}
	if req.Filename == "" || req.MimeType == "" || req.Size <= 0 {
		return nil, fmt.Errorf("invalid upload request: missing required fields")
	}

	resp, err := s.httpClient.Post("uploads", "application/json", payload)
	if err != nil {
		return nil, fmt.Errorf("failed to POST upload metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]string
		err := json.NewDecoder(resp.Body).Decode(&errResp)
		errMsg := errResp["error"]
		if err != nil || errMsg == "" {
			errMsg = "unknown upstream error or invalid JSON response"
		}
		slog.WarnContext(ctx, "failed to fetch upload metadata from backend", "status", resp.Status, "error", errMsg)
		return nil, fmt.Errorf("backend error (status %d): %s", resp.StatusCode, errMsg)
	}

	var metadata map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("failed to decode upload metadata: %w", err)
	}

	return metadata, nil
}
