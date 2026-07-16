Returns `s` re-wrapped to lines of at most `width` columns. Whitespace is the only break point; existing newlines are preserved. Words longer than `width` are not split: they overflow on their own line, matching the standard Unix `fmt(1)` behaviour and what every editor's word-wrap mode does.

Width is counted in Unicode codepoints; this function is not aware of terminal-cell width for double-width East-Asian characters or zero-width modifiers. For terminal layout that depends on visual width, post-process with a width-aware library.

Backed by [`github.com/mitchellh/go-wordwrap`](https://github.com/mitchellh/go-wordwrap).