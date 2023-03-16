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
	tests := map[string]struct {
		name           string
		url            string
		wantUser       string
		wantHeader     map[string]string
		wantStatusCode int
		wantCacheKey   string
	}{
		"success": {
			name:     "Testing 200 response",
			url:      "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser: "Link-",
			wantHeader: map[string]string{
				"Link": "<https://api.github.com/user/12345/starred?page=2&per_page=1>; rel=\"next\", <https://api.github.com/user/12345/starred?page=843&per_page=1>; rel=\"last\"",
			},
			wantStatusCode: 200,
			wantCacheKey:   "2d06a89b2687745713ef0f025b8fff17873b870e7304300a982286816e471e6e",
		},
		"api_rate_limit_reached": {
			name:     "Testing 403 response",
			url:      "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser: "Link-",
			wantHeader: map[string]string{
				"X-RateLimit-Used":      "5000",
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     "1630000000",
			},
			wantStatusCode: 403,
			wantCacheKey:   "",
		},
		"user_not_found": {
			name:           "Testing 404 response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{},
			wantStatusCode: 404,
			wantCacheKey:   "",
		},
	}

	// Testing 200 response
	mockClient := NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), tests["success"].url)
		response := http.Response{
			StatusCode: tests["success"].wantStatusCode,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
		for k, v := range tests["success"].wantHeader {
			response.Header.Set(k, v)
		}
		return &response
	})
	got, err := GenerateCacheKey(tests["success"].wantUser, mockClient)
	gotCacheKey := fmt.Sprintf("%x", got)
	assert.NoError(t, err)
	assert.Equal(t, tests["success"].wantCacheKey, gotCacheKey)

	// Testing 403 response
	mockClient = NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), tests["api_rate_limit_reached"].url)
		response := http.Response{
			StatusCode: tests["api_rate_limit_reached"].wantStatusCode,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
		for k, v := range tests["api_rate_limit_reached"].wantHeader {
			response.Header.Set(k, v)
		}
		return &response
	})
	got, err = GenerateCacheKey(tests["api_rate_limit_reached"].wantUser, mockClient)
	assert.Error(t, err)
	assert.Equal(t, [32]byte{}, got)

	// Testing 404 response
	mockClient = NewTestClient(func(req *http.Request) *http.Response {
		assert.Equal(t, req.URL.String(), tests["user_not_found"].url)
		response := http.Response{
			StatusCode: tests["user_not_found"].wantStatusCode,
			Body:       ioutil.NopCloser(bytes.NewBufferString(`OK`)),
			Header:     make(http.Header),
		}
		for k, v := range tests["user_not_found"].wantHeader {
			response.Header.Set(k, v)
		}
		return &response
	})
	got, err = GenerateCacheKey(tests["user_not_found"].wantUser, mockClient)
	assert.Error(t, err)
	assert.Equal(t, [32]byte{}, got)
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
