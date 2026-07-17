<!-- Edit here: this is the MarkdownDescription source for the burnham wrap function. docs/functions/wrap.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `s` re-wrapped so lines break at whitespace to stay within `width` columns where possible. Whitespace is the only break point; existing newlines are preserved. The one exception is a word longer than `width`: it is not split and overflows on its own line, matching the standard Unix `fmt(1)` behaviour and what every editor's word-wrap mode does.

Width is counted in Unicode codepoints; this function is not aware of terminal-cell width for double-width East-Asian characters or zero-width modifiers. For terminal layout that depends on visual width, post-process with a width-aware library.

Backed by [`github.com/mitchellh/go-wordwrap`](https://github.com/mitchellh/go-wordwrap).