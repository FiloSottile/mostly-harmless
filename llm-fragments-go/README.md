# llm-hacker-news

LLM plugin for pulling Go package docs with `go doc -all`.

For background on llm fragments see [Simon Willison's blog](https://simonwillison.net/2025/Apr/7/long-context-llm/).

## Installation

Install this plugin in the same environment as [LLM](https://llm.datasette.io/).

```bash
llm install llm-fragments-go
```

## Usage

You can feed the docs of a Go package into LLM using the `go:` [fragment](https://llm.datasette.io/en/stable/fragments.html) with the package name, optionally followed by a version suffix.

```bash
llm -f go:golang.org/x/mod/sumdb/note@v0.23.0 "Write a single file command that generates a key, prints the verifier key, signs an example message, and prints the signed note."
```
