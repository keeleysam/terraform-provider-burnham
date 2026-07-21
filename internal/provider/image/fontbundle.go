package image

import (
	"regexp"
	"strings"

	monobold "github.com/go-fonts/liberation/liberationmonobold"
	monobolditalic "github.com/go-fonts/liberation/liberationmonobolditalic"
	monoitalic "github.com/go-fonts/liberation/liberationmonoitalic"
	monoregular "github.com/go-fonts/liberation/liberationmonoregular"
	sansbold "github.com/go-fonts/liberation/liberationsansbold"
	sansbolditalic "github.com/go-fonts/liberation/liberationsansbolditalic"
	sansitalic "github.com/go-fonts/liberation/liberationsansitalic"
	sansregular "github.com/go-fonts/liberation/liberationsansregular"
	serifbold "github.com/go-fonts/liberation/liberationserifbold"
	serifbolditalic "github.com/go-fonts/liberation/liberationserifbolditalic"
	serifitalic "github.com/go-fonts/liberation/liberationserifitalic"
	serifregular "github.com/go-fonts/liberation/liberationserifregular"
)

// bundledFonts are loaded into resvg's fontdb on every render. The Noto variable
// families back the CSS generics (mapped in the wasm shim); the Liberation
// families are metric-compatible with Arial / Times New Roman / Courier New and
// back the web-safe alias map below; Noto Color Emoji renders color emoji.
var bundledFonts = [][]byte{
	notoSans, notoSansItalic, notoSerif, notoSerifItalic, notoSansMono, notoColorEmoji,
	sansregular.TTF, sansbold.TTF, sansitalic.TTF, sansbolditalic.TTF,
	serifregular.TTF, serifbold.TTF, serifitalic.TTF, serifbolditalic.TTF,
	monoregular.TTF, monobold.TTF, monoitalic.TTF, monobolditalic.TTF,
}

// webSafeAliases maps common named fonts (lowercased) to a bundled family.
// resvg has no font-alias table, so we rewrite font-family before rendering.
// Arial / Times / Courier map to the metric-compatible Liberation families;
// other well-known names map to the right category so, e.g., "Georgia" renders
// serif rather than falling back to the sans default.
var webSafeAliases = map[string]string{
	"arial":           "Liberation Sans",
	"arial narrow":    "Liberation Sans",
	"helvetica":       "Liberation Sans",
	"helvetica neue":  "Liberation Sans",
	"verdana":         "Liberation Sans",
	"tahoma":          "Liberation Sans",
	"trebuchet ms":    "Liberation Sans",
	"segoe ui":        "Liberation Sans",
	"times new roman": "Liberation Serif",
	"times":           "Liberation Serif",
	"georgia":         "Liberation Serif",
	"courier new":     "Liberation Mono",
	"courier":         "Liberation Mono",
	"consolas":        "Liberation Mono",
	"monaco":          "Liberation Mono",
	"menlo":           "Liberation Mono",
}

var (
	famAttrDouble = regexp.MustCompile(`(?i)font-family\s*=\s*"([^"]*)"`)
	famAttrSingle = regexp.MustCompile(`(?i)font-family\s*=\s*'([^']*)'`)
	famCSS        = regexp.MustCompile(`(?i)font-family\s*:\s*([^;"'}]*)`)
)

// aliasFontFamilies rewrites font-family values in the SVG (both the presentation
// attribute and CSS/style forms) so known named fonts resolve to a bundled
// family. Comma-separated fallback lists are remapped token by token; unknown
// names are left untouched (they fall back to the default family in resvg).
func aliasFontFamilies(svg string) string {
	remap := func(list string) string {
		parts := strings.Split(list, ",")
		for i, p := range parts {
			token := strings.TrimSpace(p)
			key := strings.ToLower(strings.Trim(token, `"'`))
			if fam, ok := webSafeAliases[strings.TrimSpace(key)]; ok {
				parts[i] = fam
			} else {
				parts[i] = token
			}
		}
		return strings.Join(parts, ", ")
	}
	svg = famAttrDouble.ReplaceAllStringFunc(svg, func(m string) string {
		return `font-family="` + remap(famAttrDouble.FindStringSubmatch(m)[1]) + `"`
	})
	svg = famAttrSingle.ReplaceAllStringFunc(svg, func(m string) string {
		return `font-family="` + remap(famAttrSingle.FindStringSubmatch(m)[1]) + `"`
	})
	svg = famCSS.ReplaceAllStringFunc(svg, func(m string) string {
		return "font-family:" + remap(famCSS.FindStringSubmatch(m)[1])
	})
	return svg
}
