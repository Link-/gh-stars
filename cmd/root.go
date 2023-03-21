package cmd

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

var (
	user      string
	find      string
	cacheFile string
	limit     int
	verbose   bool
	version   bool
	debug     bool
	// TODO: replace with a logging library
	Client      *http.Client
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
		Client = &http.Client{}
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
		a, err := GenerateCacheKey(user)
		if err != nil {
			ErrorLogger.Fatal("Not able to generate a cache key", err)
		}
		fmt.Printf("%x", a)

	},
}

/**
 * Every API call to GitHub returns a header Link. This header contains
 * the URL to the next & last pages of results.
 * If we make a call to the API endpoint with 1 item per page, we will receive
 * a Link header with the total number of pages equal to the total number of items.
 * We can use this to generate a cache key that will be unique to the user and
 * the number of items they have starred.
 * When the user adds or removes an item, the cache key will change.
 *
 * Caveat:
 * 	if the user has starred an item then unstarred another item, the cache key
 * 	will not change! This is an acceptable tradeoff for the simplicity of the
 * 	implementation.
 **/
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

	resp, err := Client.Do(req)
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

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func GetStarredRepos(user string, cacheFile string, cacheKey [32]byte) (bytes.Buffer, error) {
	// Check if the cache file exists
	// If it does, check if the cache key is the same as the one we generated
	// If it is, return the cached data
	// If it isn't, make the API call and update the cache file
	// If it doesn't, make the API call and create the cache file
	cacheFileExists := fileExists(cacheFile)
	if cacheFileExists {
		// TODO: replace with a function that will read the content of the cache file
		return bytes.Buffer{}, nil
	}

	// Create a cachefile named: gh-stars/stars_1das3423.json
	InfoLogger.Println("Cache file doesn't exist, creating a new one")
	path := filepath.Join(os.TempDir(), fmt.Sprintf("stars_%x.json", cacheKey[:6]))
	fmt.Println(path)
	InfoLogger.Println("Cache file created: ", path)

	return bytes.Buffer{}, nil

	// args := []string{"api", fmt.Sprintf("users/%v/starred?page=1&per_page=1", user)}
	// stdOut, _, err := gh.Exec(args...)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return *bytes.NewBuffer([]byte{}), err
	// }
	// return stdOut, nil
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
