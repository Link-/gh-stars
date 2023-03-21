package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

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

func TestGenerateCacheKey(t *testing.T) {
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
			wantStatusCode: 404,
			wantCacheKey:   "",
		},
		{
			name:           "Testing 403 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{"X-RateLimit-Used": "5000", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1630000000"},
			wantStatusCode: 403,
			wantCacheKey:   "",
		},
		{
			name:           "Testing 200 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{"Link": "<https://api.github.com/user/12345/starred?page=2&per_page=1>; rel=\"next\", <https://api.github.com/user/12345/starred?page=843&per_page=1>; rel=\"last\""},
			wantStatusCode: 200,
			wantCacheKey:   "2d06a89b2687745713ef0f025b8fff17873b870e7304300a982286816e471e6e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := NewTestClient(func(req *http.Request) *http.Response {
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
			got, err := GenerateCacheKey(tt.wantUser, mockClient)
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


// func TestGetStarredRepos(t *testing.T) {
// 	// var parsedResponse []map[string]any
// 	_, err := GetStarredRepos("Link-")
// 	if err != nil {
// 		t.Errorf("Failed to get starred repos for user Link-: %v", err)
// 	}
// 	// if err := json.Unmarshal(got.Bytes(), &parsedResponse); err != nil {
// 	// 	fmt.Print(err)
// 	// }
// }
