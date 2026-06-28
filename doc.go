// Package searchwire is a zero-configuration Go metasearch runtime for agent tooling.
// Callers provide only a search query; built-in sources fan out concurrently,
// partial failures are reported in the response, and results are deduplicated and ranked.
package searchwire
