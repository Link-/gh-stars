```sh
          *      $$\                  *                 
                 $$ |     *                     *        
       $$$$$$$\ $$$$$$\    $$$$$$\   $$$$$$\   $$$$$$$\ 
     $$  _____|\_$$  _|   \____$$\ $$  __$$\ $$  _____|
  *  \$$$$$$\ *  $$ |     $$$$$$$ |$$ |  \__|\$$$$$$\  
      \____$$\   $$ |$$\ $$  __$$ |$$ |       \____$$\     *
     $$$$$$$  |  \$$$$  |\$$$$$$$ |$$ |  *   $$$$$$$  |
     \_______/ *  \____/  \_______|\__|      \_______/ 
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

  -c, --cache-file <file path>
    File you want to store the cache file in. If not provided, the tool will generate one in $TMPDIR

  -f, --find <keyword>
    The keyword you want to search for. Example: es6

  -l, --limit <number>
    Limit the search results to the specified number. Default is 10

  -v, --version
    Outputs release version

  -d, --debug
    Outputs debugging log
```

### Examples

**Simple search**

```sh
# Search for es6 in Link-'s starred repositories
gh stars -u Link- -f es6
```

```text
TODO: add response
```

**Override cache directory:**

```sh
gh stars --user 'link-' --cache-file '/tmp/.cache' --find 'es6'
```

```text
TODO: add response
```

## LICENSE

TODO: add license