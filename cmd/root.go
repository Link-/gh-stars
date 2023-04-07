package cmd

import (
	"bytes"
	"container/heap"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Link-/gh-stars/lib/pq"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

type Repo struct {
	Name      string `json:"name"`
	Full_name string `json:"full_name"`
	Private   bool   `json:"private"`
	Owner     struct {
		Login string `json:"login"`
		Url   string `json:"url"`
	}
	Description string   `json:"description"`
	Fork        bool     `json:"fork"`
	Stars       int      `json:"stargazers_count"`
	Topics      []string `json:"topics"`
}

type githubInterface interface {
	Exec(args ...string) (bytes.Buffer, bytes.Buffer, error)
}

type github struct{}

func (g *github) Exec(args ...string) (bytes.Buffer, bytes.Buffer, error) {
	return gh.Exec(args...)
}

var (
	user      string
	find      string
	cacheFile string
	limit     int
	verbose   bool
	version   bool
	debug     bool

	gh          githubInterface
	client      *http.Client
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "gh stars",
	Short: "gh stars: Search starred repositories on GitHub",
	Long:  "gh stars: Search your or any other user's starred repositories on GitHub for a keyword",
	PreRun: func(cmd *cobra.Command, args []string) {
		// Push info logs to stdout only if debug mode is enabled
		// otherwise discard it. I don't want to manage conditionals all over the place, having
		// multiple loggers is the way to go
		logWriter := io.Discard
		if debug {
			logWriter = os.Stdout
		}
		InfoLogger = log.New(logWriter, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
		ErrorLogger = log.New(os.Stderr, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
		// Initialize the HTTP client
		client = &http.Client{}
		// Initialize the GitHub client
		gh = &github{}
	},
	Run: func(cmd *cobra.Command, args []string) {
		if version {
			fmt.Printf("gh star v%s", VERSION)
			os.Exit(1)
		}
		InfoLogger.Println("Debug mode is enabled")
		InfoLogger.Println("Parameters provided ", strings.Join(os.Args[1:], " "))
		if user == "" || find == "" {
			ErrorLogger.Fatal("The --user, -u and --find, -f flags are required. See --help for more information")
		}
		// The Link header is unique enough to generate a cache key
		key, err := GenerateCacheKey(user)
		if err != nil {
			ErrorLogger.Fatal("Not able to generate a cache key", err)
		}
		starred, err := GetStarredRepos(user, key)
		if err != nil {
			ErrorLogger.Fatal("Not able to get starred repos", err)
		}
		found, err := Search(starred, find)
		if err != nil {
			ErrorLogger.Fatal("Not able to search starred repos", err)
		}
		// Print the results
		for found.Len() > 0 {
			item := heap.Pop(&found).(*pq.Item)
			fmt.Printf("%.2d:%s ", item.Priority, item.Value)
		}
	},
}

// Find the search term in the starred repos
// Returns a priority queue with the results sorted by rank (the higher the rank, the more accurate the match)
func Search(starredRepos bytes.Buffer, find string) (pq.PriorityQueue, error) {
	var found = make(pq.PriorityQueue, 0)
	heap.Init(&found)

	var repos []Repo
	needles := strings.Fields(find)
	err := json.Unmarshal(starredRepos.Bytes(), &repos)
	if err != nil {
		return nil, err
	}

	for _, repo := range repos {
		for _, needle := range needles {
			inNameRank := fuzzy.RankMatchNormalizedFold(needle, repo.Name)
			inNameMatch := func() int {
				if fuzzy.MatchNormalizedFold(find, repo.Name) {
					return 1
				} else {
					return 0
				}
			}()
			inDescriptionRank := fuzzy.RankMatchNormalizedFold(needle, repo.Description)
			inDescriptionMatch := func() int {
				if fuzzy.MatchNormalizedFold(needle, repo.Description) {
					return 1
				} else {
					return 0
				}
			}()
			inTopicsMatch := len(fuzzy.Find(needle, repo.Topics))

			// If the needle is found in the name, description or topics, add it to the results
			// The rank priority is (the higher the score, the more accurate the match):
			// 1. Exact match in name
			// 2. Fuzzy match in name
			// 3. Exact match in description
			// 4. Fuzzy match in description
			// 5. Fuzzy match in topics
			if (inNameRank >= 0 && inNameMatch == 1) || (inDescriptionRank >= 0 && inDescriptionMatch == 1) || inTopicsMatch > 0 {
				rank := inNameMatch*10 + 1/(inNameRank+2) + inDescriptionMatch*8 + 1/(inDescriptionRank+2) + inTopicsMatch*5
				InfoLogger.Printf("Found %s in %s | rank: %d", needle, repo.Full_name, rank)
				heap.Push(&found, &pq.Item{
					Value:    repo,
					Priority: rank,
				})
				break
			} else {
				continue
			}
		}
	}

	return found, nil
}

// Every API call to GitHub returns a header Link. This header contains
// the URL to the next & last pages of results.
// If we make a call to the API endpoint with 1 item per page, we will receive
// a Link header with the total number of pages equal to the total number of items.
// We can use this to generate a cache key that will be unique to the user and
// the number of items they have starred. When the user adds or removes an item,
// the cache key will change.
//
// Caveat:
// if the user has starred an item then unstarred another item, the cache key
// will not change! This is an acceptable tradeoff for the simplicity of the
// implementation.
func GenerateCacheKey(user string) ([32]byte, error) {
	if user == "" {
		return [32]byte{}, fmt.Errorf("user cannot be empty, the implementation is faulty.")
	}

	InfoLogger.Println("Attempting to fetch the total number of starred repos for user", user)
	url := fmt.Sprintf("https://api.github.com/users/%v/starred?page=1&per_page=1", user)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return [32]byte{}, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return [32]byte{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusForbidden:
		return [32]byte{}, fmt.Errorf("API rate limit reached. Used: %v, Remaining: %v, Reset Time: %v", resp.Header.Get("X-RateLimit-Used"), resp.Header.Get("X-RateLimit-Remaining"), resp.Header.Get("X-RateLimit-Reset"))
	case http.StatusNotFound:
		return [32]byte{}, fmt.Errorf("User not found or you're not authorized to access this data.")
	case http.StatusOK:
		break
	default:
		return [32]byte{}, fmt.Errorf("Unexpected HTTP status code: %d", resp.StatusCode)
	}

	header := resp.Header.Get("Link")
	cacheKey := sha256.Sum256([]byte(header))
	InfoLogger.Println("CacheKey generated:", fmt.Sprintf("%x", cacheKey))
	return cacheKey, nil
}

// GetCachePath returns the path to the cache file to use for storing starred repos. If
// the cache file path was not provided as input, it will create a new one.
// The cache file created uses the first 6 bytes of the cache key to generate a unique
// filename.
//
// Example: <tmpdir>/stars_2d06a89b2687.json
func GetCachePath(cacheKey [32]byte) (string, error) {
	// We check if cacheFile is provided as input by the user
	if cacheFile != "" {
		InfoLogger.Println("Cache file provided as input:", cacheFile)
		return cacheFile, nil
	}

	if cacheKey == [32]byte{} {
		return "", fmt.Errorf("cacheKey cannot be empty, the implementation is faulty.")
	}

	// cacheFile format: <tmpdir>/stars_2d06a89b2687.json
	// each byte is 2 hex characters
	path := filepath.Join(os.TempDir(), fmt.Sprintf("stars_%x.json", cacheKey[:6]))
	if cacheFileExists := fileExists(path); !cacheFileExists {
		InfoLogger.Println("Cache file doesn't exist, creating a new one at:", path)
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return "", err
		}
		defer file.Close()
	}
	return path, nil
}

// GetStarredRepos returns the starred repos for the given user.
// If the cache file exists and is not empty, it will read from the cache file.
// If the cache file does not exist or is empty, it will make an API call to GitHub
// to fetch the starred repos for the given user.
func GetStarredRepos(user string, cacheKey [32]byte) (bytes.Buffer, error) {
	path, err := GetCachePath(cacheKey)
	if err != nil {
		return bytes.Buffer{}, err
	}

	size, err := fileSize(path)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Read from cache file if it exists and is not empty
	if size > 0 {
		InfoLogger.Println("Cache file exists and is not empty, reading from the cache file:", path)
		file, err := os.Open(path)
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
	}

	// Cache file is empty, make an API call to GitHub and cache the results
	// TODO: pagination
	InfoLogger.Println("Cache is empty. Fetching the starred repos for:", user)
	args := []string{"api", fmt.Sprintf("users/%v/starred?page=1&per_page=1", user)}
	stdOut, _, err := gh.Exec(args...)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Write stdOut to the cache file
	InfoLogger.Println("Writing the fetched repos to cache.")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return bytes.Buffer{}, err
	}
	defer file.Close()

	_, err = file.Write(stdOut.Bytes())
	if err != nil {
		return bytes.Buffer{}, err
	}

	return stdOut, nil
}

// Checks if a file exists at the given path
func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

// Returns the size of the file at the given path.
// Returns -1 if the file does not exist.
func fileSize(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return -1, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}

func init() {
	// 	Options:
	//   -h, --help
	//     Show this message and exit.
	//   -u, --user <handle>
	//     Any GitHub handle. Example: link-
	//   -c, --cache-dir <directory>
	//     Directory you want to store the cache file in. Example: /tmp/.starscache
	//   -f, --find <keyword>
	//     The keyword you want to search for. Example: es6
	//   -l, --limit <number>
	//     Limit the search results to the specified number. Default is 10
	//   -V, --verbose
	//     Print activity log
	//   -v, --version
	//     Print current version
	//   -d, --debug
	//     Enables debug mode
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "GitHub handle of the user you want to search their stars (required)")
	rootCmd.Flags().StringVarP(&find, "find", "f", "", "The keyword you want to search for (required)")
	rootCmd.Flags().StringVarP(&cacheFile, "cache-file", "c", "", "File you want to store the cache file in. If not provided, the tool will generate one in $TMPDIR/.starscache")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Limit the search results to the specified number, default: 10")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "V", false, "Print activity log, default: false")
	rootCmd.Flags().BoolVarP(&version, "version", "v", false, "Print current version")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "Enables debug mode, default: false")
	rootCmd.SetHelpTemplate(getRootHelp())
}

func getRootHelp() string {

	return `
          *      $$\                  *                 
                 $$ |     *                     *        
       $$$$$$$\ $$$$$$\    $$$$$$\   $$$$$$\   $$$$$$$\ 
     $$  _____|\_$$  _|   \____$$\ $$  __$$\ $$  _____|
  *  \$$$$$$\ *  $$ |     $$$$$$$ |$$ |  \__|\$$$$$$\  
      \____$$\   $$ |$$\ $$  __$$ |$$ |       \____$$\     *
     $$$$$$$  |  \$$$$  |\$$$$$$$ |$$ |  *   $$$$$$$  |
     \_______/ *  \____/  \_______|\__|      \_______/ 

Fast full-text search for a keyword in your or any other user's GitHub starred repositories.
Complete documentation is available at: https://github.com/Link-/gh-stars

Synoposis:
	gh stars -u <handle> -f <keyword> [flags]

Usage:
	gh stars -u <handle> -f <keyword>

	You can search for a keyword in a user's starred repositories, these 2 flags are required.

Flags:

	Required:
	-u, --user <handle>         Any GitHub handle, e.g. Link-
	-f, --find <keyword>        The keyword you want to search for, e.g. es6

	Optional:
	-c, --cache-dir <directory> Directory you want to store the cache file in, e.g. /tmp/.starscache
	-l, --limit <number>        Limit the search results to the specified number, e.g. 10
	-V, --verbose               Outputs debugging log
	-v, --version               Outputs release version
	-d, --debug                 Outputs stack trace in case an exception is thrown

Examples:

	# Search for es6 in Link-'s starred repositories
	gh stars -u Link- -f es6

	# Limit the results to 5
	gh stars -u Link- -f es6 -l 5

	# Store the cache file in /tmp/.starscache
	gh stars -u Link- -f es6 -c /tmp/.starscache

	# Print activity log
	gh stars -u Link- -f es6 -V

	# Enable debug mode
	gh stars -u Link- -f es6 -d

	# Print current version
	gh stars -v
`
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		ErrorLogger.Fatal(err)
		os.Exit(1)
	}
}
