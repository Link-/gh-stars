package cmd

import (
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

var (
	user        string
	find        string
	cacheDir    string
	limit       int
	verbose     bool
	version     bool
	debug       bool
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	client      *http.Client
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
		a, err := GenerateCacheKey(user, client)
		if err != nil {
			ErrorLogger.Fatal("Not able to generate a cache key", err)
		}
		fmt.Printf("%x", a)

	},
}

/**
 * Every API call to GitHub returns a header Link. This header contains
 * the URL to the next page of results.
 * If we make a call to the API endpoint with 1 item per page, we will receive
 * a Link header with the total number of pages equal to the total number of items.
 * We can use this to generate a cache key that will be unique to the user and
 * the number of items they have starred.
 * When the user adds or removes an item, the cache key will change.
 **/
func GenerateCacheKey(user string, client *http.Client) ([32]byte, error) {
	if user == "" {
		return [32]byte{}, fmt.Errorf("user cannot be empty")
	}
	resourceUrl := fmt.Sprintf("https://api.github.com/users/%v/starred?page=1&per_page=1", user)
	req, err := http.NewRequest("GET", resourceUrl, nil)
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
	switch resp.StatusCode {
	case 403:
		return [32]byte{}, fmt.Errorf("API rate limit reached. Used: %v, Remaining: %v, Reset Time: %v", resp.Header.Get("X-RateLimit-Used"), resp.Header.Get("X-RateLimit-Remaining"), resp.Header.Get("X-RateLimit-Reset"))
	case 404:
		return [32]byte{}, fmt.Errorf("User not found or you're not authorized to access this data.")
	}
	defer resp.Body.Close()
	header := resp.Header.Get("Link")
	cacheKey := sha256.Sum256([]byte(header))
	return cacheKey, nil
}

func GetStarredRepos(user string) ([]byte, error) {
	// TODO: add pagination support
	// opts := api.ClientOptions{
	// 	Host: "https://api.github.com",
	// 	Headers: map[string]string{
	// 		"Accept":               "application/vnd.github+json",
	// 		"X-GitHub-Api-Version": "2022-11-28",
	// 	},
	// 	EnableCache: true,
	// }
	// client, err := gh.RESTClient(&opts)
	// resourceUrl := fmt.Sprintf("users/%v/starred?page=1&per_page=1", user)
	// resp, err := client.Request("GET", resourceUrl, nil)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer resp.Body.Close()
	// headers, err := resp.Header.Get("X-Cache-Status")
	// body, err := io.ReadAll(resp.Body)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// return body, nil
	return nil, nil
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
	rootCmd.Flags().StringVarP(&cacheDir, "cache-dir", "c", "/tmp/.starscache", "Directory you want to store the cache file in, default: /tmp/.starscache")
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
