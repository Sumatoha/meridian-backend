package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"time"

	"github.com/google/uuid"
)

// Client handles Supabase Storage operations.
type Client struct {
	httpClient  *http.Client
	storageURL  string
	serviceKey  string
	bucketName  string
}

// NewClient creates a new Supabase Storage client.
func NewClient(storageURL, serviceKey string) *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 120 * time.Second},
		storageURL:  storageURL,
		serviceKey:  serviceKey,
		bucketName:  "media",
	}
}

// Upload stores a file and returns the storage path.
func (c *Client) Upload(ctx context.Context, userID uuid.UUID, filename string, contentType string, data io.Reader) (string, error) {
	ext := path.Ext(filename)
	storagePath := fmt.Sprintf("%s/%s%s", userID.String(), uuid.New().String(), ext)

	body, err := io.ReadAll(data)
	if err != nil {
		return "", fmt.Errorf("storage: read upload data: %w", err)
	}

	url := fmt.Sprintf("%s/object/%s/%s", c.storageURL, c.bucketName, storagePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("storage: create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage: upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("storage: upload error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return storagePath, nil
}

// GetPublicURL returns a public URL for a stored file.
func (c *Client) GetPublicURL(storagePath string) string {
	return fmt.Sprintf("%s/object/public/%s/%s", c.storageURL, c.bucketName, storagePath)
}

// Delete removes a file from storage.
func (c *Client) Delete(ctx context.Context, storagePath string) error {
	url := fmt.Sprintf("%s/object/%s/%s", c.storageURL, c.bucketName, storagePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("storage: create delete request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.serviceKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage: delete: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("storage: delete error (status %d)", resp.StatusCode)
	}

	return nil
}
