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
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Link-/gh-stars/lib/pq"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/tableprinter"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/spf13/cobra"
)

const VERSION = "0.1.1"
const MAX_FUZZY_DISTANCE = 2 // Maximum Levenshtein distance for fuzzy search. Higher values are more permissive

type Repo struct {
	Name      string `json:"name"`
	Full_name string `json:"full_name"`
	Private   bool   `json:"private"`
	Url       string `json:"html_url"`
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
	user       string
	find       string
	cacheFile  string
	limit      int
	version    bool
	jsonOutput bool
	debug      bool

	ghClient    githubInterface
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
		ghClient = &github{}
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

		// Generate the cache key from the Link header
		key, err := GenerateCacheKey(user)
		if err != nil {
			ErrorLogger.Fatal("Not able to generate a cache key", err)
		}

		// Pull the starred repos from the cache or from the API if the cache is empty
		starred, err := GetStarredRepos(user, key)
		if err != nil {
			ErrorLogger.Fatal("Not able to get starred repos", err)
		}

		// Fuzzy and ranked searched for the search term(s)
		found, err := Search(starred, find)
		if err != nil {
			ErrorLogger.Fatal("Not able to search starred repos", err)
		}

		if err := Render(found, limit, os.Stdout); err != nil {
			ErrorLogger.Fatal("Not able to render the table", err)
		}
	},
	Version: VERSION,
}

func Render(results pq.PriorityQueue, limit int, renderTarget io.Writer) error {
	switch jsonOutput {
	case true:
		return RenderJsonOutput(results, limit, renderTarget)
	default:
		return RenderTable(results, limit, renderTarget)
	}
}

func RenderTable(results pq.PriorityQueue, limit int, renderTarget io.Writer) error {
	InfoLogger.Println("Rendering the results in table format")

	if results.Len() > limit {
		InfoLogger.Printf("Results: %d are higher than the limit: %d \n", results.Len(), limit)
	}

	renderLimit := RenderLimit(results.Len(), limit)

	tp := tableprinter.New(renderTarget, true, 350)
	headerRow := []string{"Name", "URL", "Description", "Stars", "Rank"}
	for _, item := range headerRow {
		tp.AddField(item)
	}
	tp.EndRow()
	for i := 0; i < renderLimit; i++ {
		item := heap.Pop(&results).(*pq.Item)
		tp.AddField(item.Value.(Repo).Full_name)
		tp.AddField(item.Value.(Repo).Url)
		tp.AddField(item.Value.(Repo).Description)
		tp.AddField(fmt.Sprintf("%d", item.Value.(Repo).Stars))
		tp.AddField(fmt.Sprintf("%d", item.Priority))
		tp.EndRow()
	}
	err := tp.Render()
	if err != nil {
		return err
	}
	return nil
}

// RenderJsonOutput renders the results in JSON format
func RenderJsonOutput(results pq.PriorityQueue, limit int, renderTarget io.Writer) error {
	InfoLogger.Println("Rendering the results in JSON format")

	if results.Len() > limit {
		InfoLogger.Printf("Results: %d are higher than the limit: %d \n", results.Len(), limit)
	}

	renderLimit := RenderLimit(results.Len(), limit)

	var repos []Repo
	for i := 0; i < renderLimit; i++ {
		item := heap.Pop(&results).(*pq.Item)
		repos = append(repos, item.Value.(Repo))
	}

	jsonOutput, err := json.Marshal(repos)
	if err != nil {
		return err
	}

	fmt.Fprintf(renderTarget, "%s", jsonOutput)
	return nil
}

// RenderLimit returns the limit to be used for rendering the results
// If the limit is -1, then return the total number of results
// Otherwise return the minimum of the limit and the total number of results
func RenderLimit(resultsCount int, limit int) int {
	if limit <= -1 {
		return resultsCount
	} else {
		return int(math.Min(float64(limit), float64(resultsCount)))
	}
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
			// Handle the repository name
			// Split the repository on - and _
			repoNameWords := strings.FieldsFunc(repo.Name, func(r rune) bool {
				return r == '-' || r == '_'
			})
			match := false
			for _, word := range repoNameWords {
				rank := fuzzy.LevenshteinDistance(needle, word)
				if rank >= 0 && rank <= MAX_FUZZY_DISTANCE {
					heap.Push(&found, &pq.Item{
						Value:    repo,
						Priority: (rank/(rank+1) + 10) * 100,
					})
					match = true
					break
				}
			}
			if match {
				continue
			}
			// Handle the repository description
			descriptionWords := strings.Fields(repo.Description)
			for _, word := range descriptionWords {
				rank := fuzzy.LevenshteinDistance(needle, word)
				if rank >= 0 && rank <= MAX_FUZZY_DISTANCE {
					heap.Push(&found, &pq.Item{
						Value:    repo,
						Priority: (rank/(rank+1) + 5) * 50,
					})
				}
				continue
			}
			// Handle the topics
			for _, topic := range repo.Topics {
				rank := fuzzy.LevenshteinDistance(needle, topic)
				if rank >= 0 && rank <= MAX_FUZZY_DISTANCE {
					heap.Push(&found, &pq.Item{
						Value:    repo,
						Priority: (rank/(rank+1) + 1) * 25,
					})
				}
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
	InfoLogger.Println("Cache is empty. Fetching the starred repos for:", user)
	args := []string{"api", "--paginate", fmt.Sprintf("users/%v/starred", user)}
	stdOut, _, err := ghClient.Exec(args...)
	if err != nil {
		return bytes.Buffer{}, err
	}

	// TODO: This is extremely nasty, and needs to be refactored once this PR
	// is merged: https://github.com/cli/cli/pull/7190
	// This resolves the problem of gh api --paginate returning concatenated slices instead of
	// a single slice of all the results
	result := stdOut.String()
	jsonResult := strings.Replace(result, "][", ",", -1)
	resultBuffer := bytes.NewBufferString(jsonResult)

	// Write stdOut to the cache file
	InfoLogger.Println("Writing the fetched repos to cache.")
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return bytes.Buffer{}, err
	}
	defer file.Close()

	_, err = file.Write(resultBuffer.Bytes())
	if err != nil {
		return bytes.Buffer{}, err
	}

	return *resultBuffer, nil
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
	//   -c, --cache-file <file path>
	//     File you want to store the cache in. File should exist and be writable. If not provided, the tool will generate one in $TMPDIR
	//   -f, --find <keyword>
	//     The keyword you want to search for. Example: es6
	//   -l, --limit <number>
	//     Limit the search results to the specified number. Default is 10
	//   -v, --version
	//     Print current version
	//   -d, --debug
	//     Outputs debugging log
	rootCmd.Flags().StringVarP(&user, "user", "u", "", "GitHub handle of the user you want to search their stars (required)")
	rootCmd.Flags().StringVarP(&find, "find", "f", "", "The keyword you want to search for (required)")
	rootCmd.Flags().StringVarP(&cacheFile, "cache-file", "c", "", "File you want to store the cache file in. If not provided, the tool will generate one in $TMPDIR")
	rootCmd.Flags().IntVarP(&limit, "limit", "l", 10, "Limit the search results to the specified number, default: 10")
	rootCmd.Flags().BoolVarP(&version, "version", "v", false, "Print current version")
	rootCmd.Flags().BoolVarP(&jsonOutput, "json", "j", false, "Prints the output in JSON format, default: false")
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

Fast fuzzy search for a keyword in your or any other user's GitHub starred repositories.
Complete documentation is available at: https://github.com/Link-/gh-stars

Synoposis:
	gh stars -u <handle> -f <keyword> [flags]

Usage:
	gh stars -u <handle> -f <keyword>

	You can search for a keyword in a user's starred repositories, these 2 flags are required.

Flags:

	Required:
	-u, --user <handle>          Any GitHub handle, e.g. Link-
	-f, --find <keyword>         The keyword you want to search for, e.g. es6

	Optional:
	-c, --cache-file <file path> File you want to store the cache in. File should exist and be writable. If not provided, the tool will generate one in $TMPDIR
	-l, --limit <number>         Limit the search results to the specified number, e.g. 10
	-v, --version                Outputs release version
	-j, --json				     Outputs the results in JSON format
	-d, --debug                  Outputs debugging log

Examples:

	# Search for es6 in Link-'s starred repositories
	gh stars -u Link- -f es6

	# Limit the results to 5
	gh stars -u Link- -f es6 -l 5

	# Store the cache file in /tmp/.starscache
	gh stars -u Link- -f es6 -c /tmp/.starscache

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
