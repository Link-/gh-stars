package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const VERSION = "0.1.0"

var (
	user     string
	find     string
	cacheDir string
	limit    int
	verbose  bool
	version  bool
	debug    bool
)

var rootCmd = &cobra.Command{
	Use:   "gh stars",
	Short: "gh stars: Search starred repositories on GitHub",
	Long:  "gh stars: Search your or any other user's starred repositories on GitHub for a keyword",
	Run: func(cmd *cobra.Command, args []string) {
		version, _ := cmd.Flags().GetBool("version")
		if version {
			fmt.Printf("gh star v%s", VERSION)
			os.Exit(1)
		}
		print(user, find, cacheDir, limit, verbose, debug)
	},
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
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "GitHub handle of the user you want to search their stars (required) e.g. Link-")
	rootCmd.PersistentFlags().StringVarP(&find, "find", "f", "", "The keyword you want to search for (required) e.g. es6")
	rootCmd.PersistentFlags().StringVarP(&cacheDir, "cache-dir", "c", "/tmp/.starscache", "Directory you want to store the cache file in, default: /tmp/.starscache")
	rootCmd.PersistentFlags().IntVarP(&limit, "limit", "l", 10, "Limit the search results to the specified number, default: 10")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "V", false, "Print activity log")
	rootCmd.PersistentFlags().BoolVarP(&version, "version", "v", false, "Print current version")
	rootCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "Enables debug mode")
	rootCmd.SetHelpTemplate(getRootHelp())
	_ = rootCmd.MarkFlagRequired("user")
	_ = rootCmd.MarkFlagRequired("find")

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
		fmt.Println(err)
		os.Exit(1)
	}
}
