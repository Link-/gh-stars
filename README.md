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

![Demo of how the extension works](./demo.gif)

:warning: This project is still in `alpha` and the API might change without notice. Update only after reviewing the changelog for breaking changes.

## Installation

### Setup

```text
gh extension install Link-/gh-stars
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
    File you want to store the cache in. File should exist and be writable. If not provided, the tool will generate one in $TMPDIR

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

#### Simple search

```sh
# Search for es6 in Link-'s starred repositories
gh stars --user 'link-' --find 'macos'
```

```text
Name                       URL                                           Description                                                                                                        Stars  Rank
ianyh/Amethyst             https://github.com/ianyh/Amethyst             Automatic tiling window manager for macOS à la xmonad.                                                             12815  250
jakehilborn/displayplacer  https://github.com/jakehilborn/displayplacer  macOS command line utility to configure multi-display resolutions and arrangements. Essentially XRandR for macOS.  3067   250
wailsapp/wails             https://github.com/wailsapp/wails             Create beautiful applications using Go                                                                             15933  25
ianyh/Amethyst             https://github.com/ianyh/Amethyst             Automatic tiling window manager for macOS à la xmonad.                                                             12815  25
ianyh/Amethyst             https://github.com/ianyh/Amethyst             Automatic tiling window manager for macOS à la xmonad.                                                             12815  25
massCodeIO/massCode        https://github.com/massCodeIO/massCode        A free and open source code snippets manager for developers                                                        4694   25
```

#### Override cache directory

```sh
gh stars --user 'link-' --cache-file '/tmp/.cache' --find 'markdown'
```

```text
Name                             URL                                                 Description                                                                                                                                                      Stars  Rank
MacDownApp/macdown               https://github.com/MacDownApp/macdown               Open source Markdown editor for macOS.                                                                                                                           9232   1000
evilstreak/markdown-js           https://github.com/evilstreak/markdown-js           A Markdown parser for javascript                                                                                                                                 7664   1000
charmbracelet/glamour            https://github.com/charmbracelet/glamour            Stylesheet-based markdown rendering for your CLI apps 💇🏻‍♀️                                                                                                     1620   250
Naereen/badges                   https://github.com/Naereen/badges                   :pencil: Markdown code for lots of small badges :ribbon: :pushpin: (shields.io, forthebadge.com etc) :sunglasses:. Contributions are welcome! Please add yours!  3851   250
markedjs/marked                  https://github.com/markedjs/marked                  A markdown parser and compiler. Built for speed.                                                                                                                 29694  250
ActionsDesk/report-action-usage  https://github.com/ActionsDesk/report-action-usage  Action to create a CSV or Markdown report of GitHub Actions used                                                                                                 8      250
syntax-tree/mdast                https://github.com/syntax-tree/mdast                Markdown Abstract Syntax Tree format                                                                                                                             831    250
honkit/honkit                    https://github.com/honkit/honkit                    :book: HonKit is building beautiful books using Markdown - Fork of GitBook                                                                                       2544   250
hedgedoc/hedgedoc                https://github.com/hedgedoc/hedgedoc                HedgeDoc - The best platform to write and share markdown.                                                                                                        3866   250
valentjn/vscode-ltex             https://github.com/valentjn/vscode-ltex             LTeX: Grammar/spell checker :mag::heavy_check_mark: for VS Code using LanguageTool with support for LaTeX :mortar_board:, Markdown :pencil:, and others          644    250

```

#### Debug mode enabled

```sh
gh stars --user 'link-' --find 'programming language' --debug
```

```text
INFO: 2023/05/13 21:44:42 root.go:91: Debug mode is enabled
INFO: 2023/05/13 21:44:42 root.go:92: Parameters provided  --user link- --find programming language --debug
INFO: 2023/05/13 21:44:42 root.go:239: Attempting to fetch the total number of starred repos for user link-
INFO: 2023/05/13 21:44:42 root.go:269: CacheKey generated: d856442b086a3c61b6593f22bf91804c91cbb53bfb585aa9217550f8c0271a5c
INFO: 2023/05/13 21:44:42 root.go:321: Cache file exists and is not empty, reading from the cache file: /var/folders/ld/5wnf1d_525q_ldl3kj9km5lh0000gn/T/stars_d856442b086a.json
INFO: 2023/05/13 21:44:42 root.go:124: Rendering the results
INFO: 2023/05/13 21:44:42 root.go:127: Results: 6 are higher than the limit: 10
Name                             URL                                                 Description                                                                                                                                                                  Stars  Rank
bigscience-workshop/petals       https://github.com/bigscience-workshop/petals       🌸 Run 100B+ language models at home, BitTorrent-style. Fine-tuning and inference up to 10x faster than offloading                                                           4657   250
GoogleContainerTools/distroless  https://github.com/GoogleContainerTools/distroless  🥑  Language focused docker images, minus the operating system.                                                                                                              15549  250
shyamsn97/mario-gpt              https://github.com/shyamsn97/mario-gpt              Generating Mario Levels with GPT2. Code for the paper "MarioGPT: Open-Ended Text2Level Generation through Large Language Models" https://arxiv.org/abs/2302.05981            964    250
carbon-language/carbon-lang      https://github.com/carbon-language/carbon-lang      Carbon Language's main repository: documents, design, implementation, and related tools. (NOTE: Carbon Language is experimental; see README)                                 30415  250
slimtoolkit/slim                 https://github.com/slimtoolkit/slim                 Slim(toolkit): Don't change anything in your container image and minify it by up to 30x (and for compiled languages even more) making it secure too! (free and open source)  16633  250
carbon-language/carbon-lang      https://github.com/carbon-language/carbon-lang      Carbon Language's main repository: documents, design, implementation, and related tools. (NOTE: Carbon Language is experimental; see README)                                 30415  25
```

#### Limit results

```sh
gh stars --user 'link-' --find 'a' --limit 5
```

```text
Name                      URL                                          Description                                                                                         Stars  Rank
hashicorp/go-memdb        https://github.com/hashicorp/go-memdb        Golang in-memory database built on immutable radix trees                                            2805   1000
mckaywrigley/chatbot-ui   https://github.com/mckaywrigley/chatbot-ui   An open source ChatGPT UI.                                                                          13459  1000
andyfeller/gh-montage     https://github.com/andyfeller/gh-montage     GitHub CLI extension to generate montage from GitHub user avatars                                   28     1000
actions/gh-actions-cache  https://github.com/actions/gh-actions-cache  A GitHub (gh) CLI extension to manage the GitHub Actions caches being used in a GitHub repository.  219    1000
nadrad/h-m-m              https://github.com/nadrad/h-m-m              Hackers Mind Map                                                                                    1628   1000
```

## Troubleshoot

Found a problem? [Open an issue](https://github.com/Link-/gh-stars/issues/new).