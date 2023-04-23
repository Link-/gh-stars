package cmd

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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
	// Switch to true to see the InfoLogger output
	debug = false
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
			name:           "Testing404Response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{},
			wantStatusCode: http.StatusNotFound,
			wantCacheKey:   "",
		},
		{
			name:           "Testing403Response",
			url:            "https://api.github.com/users/Link-/starred?page=1&per_page=1",
			wantUser:       "Link-",
			wantHeader:     map[string]string{"X-RateLimit-Used": "5000", "X-RateLimit-Remaining": "0", "X-RateLimit-Reset": "1630000000"},
			wantStatusCode: http.StatusForbidden,
			wantCacheKey:   "",
		},
		{
			name:           "Testing200Response",
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
			client = NewTestClient(func(req *http.Request) *http.Response {
				assert.Equal(t, req.URL.String(), tt.url)
				response := http.Response{
					StatusCode: tt.wantStatusCode,
					Body:       io.NopCloser(bytes.NewBufferString(`OK`)),
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

func TestGetCachePath(t *testing.T) {
	setup([]string{})
	tmpPath := os.TempDir()
	tests := []struct {
		name           string
		inputCacheFile string
		cacheKey       [32]byte
		wantErr        bool
		wantPath       string
	}{
		{
			name:           "InputCacheFileProvided",
			inputCacheFile: filepath.Join(tmpPath, "test.json"),
			cacheKey:       [32]byte{0x2d, 0x06, 0xa8, 0x9b, 0x26, 0x87, 0x74, 0x57, 0x13, 0xef, 0x0f, 0x02, 0x5b, 0x8f, 0xff, 0x17, 0x87, 0x3b, 0x87, 0x0e, 0x73, 0x04, 0x30, 0x0a, 0x98, 0x22, 0x86, 0x81, 0x6e, 0x47, 0x1e, 0x6e},
			wantErr:        false,
			wantPath:       filepath.Join(tmpPath, "test.json"),
		},
		{
			name:           "InputCacheFileEmpty",
			inputCacheFile: "",
			cacheKey:       [32]byte{0x2d, 0x06, 0xa8, 0x9b, 0x26, 0x87, 0x74, 0x57, 0x13, 0xef, 0x0f, 0x02, 0x5b, 0x8f, 0xff, 0x17, 0x87, 0x3b, 0x87, 0x0e, 0x73, 0x04, 0x30, 0x0a, 0x98, 0x22, 0x86, 0x81, 0x6e, 0x47, 0x1e, 0x6e},
			wantErr:        false,
			wantPath:       filepath.Join(tmpPath, "stars_2d06a89b2687.json"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Change the input package variable to the test value
			cacheFile = tt.inputCacheFile
			got, err := GetCachePath(tt.cacheKey)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantPath, got)
		})
	}
}

// MockGithub is a mock implementation of the Github interface
type MockGithub struct{}

func (m *MockGithub) Exec(args ...string) (bytes.Buffer, bytes.Buffer, error) {
	stdOut := bytes.NewBufferString("mock output")
	stdErr := bytes.NewBufferString("mock error")
	return *stdOut, *stdErr, nil
}

func TestGetStarredRepos(t *testing.T) {
	setup([]string{})

	t.Run("FetchStarredReposFromCache", func(t *testing.T) {
		// Fetches data from an existing cache file
		cacheFile = filepath.Join(os.TempDir(), "test_pull_cache.json")
		want := []byte(`{"repos": [{"name": "test-cache-file", "url": "https://github.com/test/repo"}]}`)
		file, err := os.OpenFile(cacheFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		_, err = io.Copy(file, bytes.NewBuffer(want))
		if err != nil {
			t.Fatal(err)
		}

		// Compare the cache file content to the expected data
		got, err := GetStarredRepos("Link-", [32]byte{})
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, got.Bytes())

		// Cleanup
		err = os.Remove(cacheFile)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("FetchStarredReposWithEmptyCache", func(t *testing.T) {
		// Cache file doesn't exist, so we should fetch from Github
		ghClient = &MockGithub{}
		// Make sure we're not referencing a cacheFile that exists
		cacheFile = ""
		cacheKey := [32]byte{0x2d, 0x06, 0xa8, 0x9b, 0x26, 0x87, 0x74, 0x57, 0x13, 0xef, 0x0f, 0x02, 0x5b, 0x8f, 0xff, 0x17, 0x87, 0x3b, 0x87, 0x0e, 0x73, 0x04, 0x30, 0x0a, 0x98, 0x22, 0x86, 0x81, 0x6e, 0x47, 0x1e, 0x6e}
		cachePath, err := GetCachePath(cacheKey)

		if err != nil {
			t.Fatal(err)
		}

		if fileExists(cachePath) {
			// Remove the cache file if it exists
			if err := os.Remove(cachePath); err != nil {
				t.Fatal(err)
			}
		}

		want := *bytes.NewBufferString("mock output")
		got, err := GetStarredRepos("Link-", cacheKey)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, want, got)

		if fileExists(cachePath) {
			// Remove the cache file if it exists
			if err := os.Remove(cachePath); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestSearch(t *testing.T) {
	setup([]string{})

	testData, _ := func() (bytes.Buffer, error) {
		file, err := os.Open("testdata/5_repos.json")
		if err != nil {
			return bytes.Buffer{}, err
		}
		defer file.Close()

		var data bytes.Buffer
		_, err = io.Copy(&data, file)
		if err != nil {
			return bytes.Buffer{}, err
		}
		return data, nil
	}()

	tests := []struct {
		name    string
		data    bytes.Buffer
		wantErr bool
		find    string
		pqDepth int
	}{
		{
			name:    "SearchWithEmptyTerm",
			data:    testData,
			wantErr: false,
			find:    "",
			pqDepth: 0,
		},
		{
			name:    "SearchWithSingleTerm",
			data:    testData,
			wantErr: false,
			find:    "amethyst",
			pqDepth: 1,
		},
		{
			name:    "SearchWithSingleLetterTerm",
			data:    testData,
			wantErr: false,
			find:    "y",
			pqDepth: 0,
		},
		{
			name:    "SearchWithMultipleWords",
			data:    testData,
			wantErr: false,
			find:    "amethyst engine",
			pqDepth: 1,
		},
	}

	// Run the tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Search(tt.data, tt.find)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.pqDepth, got.Len())
		})
	}
}
