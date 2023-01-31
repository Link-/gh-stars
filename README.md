```sh
✴    ✴   ✴              ✴       ✴              ✴                ✴
  ✴  _      ✴      ✴   ✴        _    ✴              ✴   ✴     _   ✴  
 ___| |_ __ _ _✴__ _ __ ___  __| |    ___ ✴___  __ _ _✴__ ___| |__  
/ __| __/ _` | '✴_| '__/ _ \/ _` |  ✴/ __|/ _ \/✴_` | '__/ __| '_ \✴
\__ \ || (✴| |✴|  | | |  __/ (_| | ✴ \__ \  __/ (_| | | | (__| |✴| |✴
|___/\__\__,_|_|  |_| ✴\___|\__,_|___|___/\___|\__✴_|_| ✴\___|_| |_|
          ✴            ✴        |_____|      ✴       ✴               
```

> Search your starred ★ repositories on GitHub from your terminal

You know those repositories you like and star into the abyss? Yes those, this CLI tool will help you do a fuzzy search on them. You can search any GitHub user's starred repositories by providing their handle only.

This tool will cache the results locally so that you don't risk abusing the API requests limit.

```text
TODO: Add demo gif
```

:warning: This project is still in `alpha` and the API might change without notice. Update only after reviewing the changelog for breaking changes.

## Installation

### Setup

```text
TODO: Add installation instructions
```

## Usage

```text
Usage: gh stars [OPTIONS] [ARGS]...

  Search your or any other user's starred repositories on GitHub for a keyword.

Options:
  -h, --help
    Show this message and exit.

  -u, --user <handle>
    Any GitHub handle. Example: link-

  -c, --cache-dir <directory>
    Directory you want to store the cache file in. Example: /tmp/.cache

  -f, --find <keyword>
    The keyword you want to search for. Example: es6

  -l, --limit <number>
    Limit the search results to the specified number. Default is 10

  -V, --verbose
    Outputs debugging log

  -v, --version
    Outputs release version

  -d, --debug
    Outputs stack trace in case an exception is thrown
```

### Examples

**Non-verbose output:**

```sh
gh stars --user 'link-' --find 'es6'
```

```json
[
  {
    "repo_name": "lukehoban/es6features",
    "repo_description": "Overview of ECMAScript 6 features",
    "repo_url": "https://github.com/lukehoban/es6features",
    "repo_stars": 27672
  },
  {
    "repo_name": "google/sa360-flightsfeed",
    "repo_description": "Generate SA360 compatible feeds for airlines on BigQuery  :rocket:",
    "repo_url": "https://github.com/google/sa360-flightsfeed",
    "repo_stars": 8
  },
  {
    "repo_name": "DrkSephy/es6-cheatsheet",
    "repo_description": "ES2015 [ES6] cheatsheet containing tips, tricks, best practices and code snippets",
    "repo_url": "https://github.com/DrkSephy/es6-cheatsheet",
    "repo_stars": 11410
  }
]
```

**Verbose output & override cache directory:**

```sh
gh stars --user 'link-' --cache-dir '/tmp/.cache' --find 'es6' --verbose
```

```json
🕵    INFO: Searching for "es6" in "link-'s" starred catalogue
⚠️    INFO:: Serving search results from cache
[
  {
    "repo_name": "lukehoban/es6features",
    "repo_description": "Overview of ECMAScript 6 features",
    "repo_url": "https://github.com/lukehoban/es6features",
    "repo_stars": 27672
  },
  {
    "repo_name": "google/sa360-flightsfeed",
    "repo_description": "Generate SA360 compatible feeds for airlines on BigQuery  :rocket:",
    "repo_url": "https://github.com/google/sa360-flightsfeed",
    "repo_stars": 8
  },
  {
    "repo_name": "DrkSephy/es6-cheatsheet",
    "repo_description": "ES2015 [ES6] cheatsheet containing tips, tricks, best practices and code snippets",
    "repo_url": "https://github.com/DrkSephy/es6-cheatsheet",
    "repo_stars": 11410
  }
]
```

**Parsing the output with jq**
You can pipe the standard output to be handled by tools like [jq](https://stedolan.github.io/jq/) for more magic:

```sh
# Return the first search result only
gh stars -u 'link-' -f 'es6' | jq '.[0]'
```

```json
{
  "repo_name": "lukehoban/es6features",
  "repo_description": "Overview of ECMAScript 6 features",
  "repo_url": "https://github.com/lukehoban/es6features",
  "repo_stars": 27672
}
```

```sh
# Return repo_name of every result element
gh stars -u 'link-' -f 'es6' | jq 'map(.repo_name)'
```

```json
[
  "lukehoban/es6features",
  "google/sa360-flightsfeed",
  "DrkSephy/es6-cheatsheet"
]
```