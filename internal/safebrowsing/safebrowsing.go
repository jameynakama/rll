// Package safebrowsing checks urls against Google Safe Browsing (v5 Lookup
// API; v4 is deprecated).
package safebrowsing

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

const DefaultBaseURL = "https://safebrowsing.googleapis.com"

// threatsField is SearchUrlsResponse.threats. v5 only answers in protobuf,
// and whether this field appears is all we need, so we read the wire format
// directly rather than pulling in generated types.
const threatsField protowire.Number = 1

type Client struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

func New(baseURL, apiKey string) *Client {
	return &Client{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 5 * time.Second},
	}
}

// IsFlagged reports whether Safe Browsing lists the url, or one of its
// host-suffix/path-prefix expressions, as a threat. It checks one url per
// request on purpose: v5 may answer with a matched expression rather than the
// url sent, so a batched response can't reliably say which url was bad.
func (c *Client) IsFlagged(ctx context.Context, rawURL string) (bool, error) {
	q := url.Values{"urls": {rawURL}, "key": {c.apiKey}}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v5/urls:search?"+q.Encode(), nil)
	if err != nil {
		return false, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("safe browsing: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("safe browsing: read: %w", err)
	}
	return hasThreats(body)
}

func hasThreats(b []byte) (bool, error) {
	for len(b) > 0 {
		num, typ, n := protowire.ConsumeTag(b)
		if n < 0 {
			return false, fmt.Errorf("safe browsing: decode: %w", protowire.ParseError(n))
		}
		if num == threatsField {
			return true, nil
		}
		b = b[n:]
		n = protowire.ConsumeFieldValue(num, typ, b)
		if n < 0 {
			return false, fmt.Errorf("safe browsing: decode: %w", protowire.ParseError(n))
		}
		b = b[n:]
	}
	return false, nil
}
