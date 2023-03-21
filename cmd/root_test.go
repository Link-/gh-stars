package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

// RoundTripFunc .
type RoundTripFunc func(req *http.Request) *http.Response

// RoundTrip .
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req), nil
}

// NewTestClient returns *http.Client with Transport replaced to avoid making real calls
func NewTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{
		Transport: RoundTripFunc(fn),
	}
}

func setup(args []string) {
	rootCmd.PreRun(&cobra.Command{}, args)
}

func TestGenerateCacheKey(t *testing.T) {
	setup([]string{})

	tests := []struct {
		name           string
		url            string
		wantUser       string
		wantHeader     map[string]string
		wantStatusCode int
		wantCacheKey   string
	}{
		{
			name:           "Testing 404 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{},
			wantStatusCode: http.StatusNotFound,
			wantCacheKey:   "",
		},
		{
			name:           "Testing 403 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{"X-RateLimit-Used": "5000", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1630000000"},
			wantStatusCode: http.StatusForbidden,
			wantCacheKey:   "",
		},
		{
			name:           "Testing 200 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{"Link": "<https://api.github.com/user/12345/starred?page=2&per_page=1>; rel=\"next\", <https://api.github.com/user/12345/starred?page=843&per_page=1>; rel=\"last\""},
			wantStatusCode: http.StatusOK,
			wantCacheKey:   "2d06a89b2687745713ef0f025b8fff17873b870e7304300a982286816e471e6e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override the client with a mock client
			Client = NewTestClient(func(req *http.Request) *http.Response {
				assert.Equal(t, req.URL.String(), tt.url)
				response := http.Response{
					StatusCode: tt.wantStatusCode,
					Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
					Header:     make(http.Header),
				}
				for k, v := range tt.wantHeader {
					response.Header.Set(k, v)
				}
				return &response
			})
			got, err := GenerateCacheKey(tt.wantUser)
			gotCacheKey := fmt.Sprintf("%x", got)
			if err != nil {
				assert.Error(t, err)
				assert.Equal(t, [32]byte{}, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCacheKey, gotCacheKey)
			}
		})
	}
}

func TestGetStarredRepos(t *testing.T) {
	cacheKey := [32]byte{0x2d, 0x06, 0xa8, 0x9b, 0x26, 0x87, 0x74, 0x57, 0x13, 0xef, 0x0f, 0x02, 0x5b, 0x8f, 0xff, 0x17, 0x87, 0x3b, 0x87, 0x0e, 0x73, 0x04, 0x30, 0x0a, 0x98, 0x22, 0x86, 0x81, 0x6e, 0x47, 0x1e, 0x6e}
	result, err := GetStarredRepos("Link-", "", cacheKey)
	fmt.Println(result)
	if err != nil {
		t.Errorf("Failed to get starred repos for user Link-: %v", err)
	}
}
