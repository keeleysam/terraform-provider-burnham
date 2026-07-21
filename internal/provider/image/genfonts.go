//go:build ignore

// Command genfonts downloads the bundled fonts that are not available as Go
// modules into fonts/, verifying each against a pinned SHA-256. It is run via
// `go generate` (see fonts.go); the downloaded .ttf files are gitignored, so a
// fresh checkout must run `go generate ./...` before the image package compiles.
//
// All fonts come from tagged upstream releases: Noto Color Emoji from
// googlefonts/noto-emoji (a bare .ttf asset) and the Noto text families from
// notofonts/latin-greek-cyrillic (zip assets, one per family, with the variable
// font inside). To update a font: bump its *Ref below, run this, and paste the
// new sha256 it reports on a mismatch.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Pinned upstream release refs.
const (
	emojiRef        = "v2.051"           // github.com/googlefonts/noto-emoji release tag
	notoSansRef     = "NotoSans-v2.015"  // github.com/notofonts/latin-greek-cyrillic tags
	notoSerifRef    = "NotoSerif-v2.015" // (per-family, tag == asset basename)
	notoSansMonoRef = "NotoSansMono-v2.014"
)

// An output is one .ttf written to disk. zipEntry is the exact path inside the
// source zip; empty means the source URL is the .ttf itself.
type output struct {
	dest, zipEntry, sha256 string
}

// A source is one download (a bare .ttf or a zip) yielding one or more outputs.
type source struct {
	url     string
	outputs []output
}

// lgcZip is the release-asset URL for a notofonts/latin-greek-cyrillic family
// tag (the asset is named "<tag>.zip").
func lgcZip(tag string) string {
	return "https://github.com/notofonts/latin-greek-cyrillic/releases/download/" + tag + "/" + tag + ".zip"
}

var manifest = []source{
	{
		url:     "https://github.com/googlefonts/noto-emoji/raw/" + emojiRef + "/fonts/Noto-COLRv1.ttf",
		outputs: []output{{dest: "fonts/Noto-COLRv1.ttf", sha256: "0ae57fe58645638523ba35f388d93739d292539a9acb84df5700c81b1e1a28d2"}},
	},
	{
		url: lgcZip(notoSansRef),
		outputs: []output{
			{dest: "fonts/NotoSans.ttf", zipEntry: "NotoSans/googlefonts/variable-ttf/NotoSans[wdth,wght].ttf", sha256: "bfb7bb691513f12e734dc346c03a03f784912432d7e3fa8e56efcf906fe86b3d"},
			{dest: "fonts/NotoSans-Italic.ttf", zipEntry: "NotoSans/googlefonts/variable-ttf/NotoSans-Italic[wdth,wght].ttf", sha256: "58e6e0ebd1931b29a365aa2d3e2ee9a9e831a3af7cf3ad1462d4e72154f0b291"},
		},
	},
	{
		url: lgcZip(notoSerifRef),
		outputs: []output{
			{dest: "fonts/NotoSerif.ttf", zipEntry: "NotoSerif/googlefonts/variable-ttf/NotoSerif[wdth,wght].ttf", sha256: "4d8e6761424656867019081a1a01336f3cb086982682698714054fc33f782713"},
			{dest: "fonts/NotoSerif-Italic.ttf", zipEntry: "NotoSerif/googlefonts/variable-ttf/NotoSerif-Italic[wdth,wght].ttf", sha256: "e9342c2b2debeee282a945e6dffde94612edd7e7b70fba9463abdb6e658ec724"},
		},
	},
	{
		url: lgcZip(notoSansMonoRef),
		outputs: []output{
			{dest: "fonts/NotoSansMono.ttf", zipEntry: "NotoSansMono/googlefonts/variable/NotoSansMono[wdth,wght].ttf", sha256: "2cb2adb378a8f574213e23df697050b83c54c27df465a2015552740b2769a081"},
		},
	},
}

func main() {
	for _, s := range manifest {
		if err := ensure(s); err != nil {
			fmt.Fprintf(os.Stderr, "genfonts: %v\n", err)
			os.Exit(1)
		}
	}
}

func ensure(s source) error {
	allOK := true
	for _, o := range s.outputs {
		if sum, err := hashFile(o.dest); err != nil || sum != o.sha256 {
			allOK = false
			break
		}
	}
	if allOK {
		return nil
	}

	body, err := download(s.url)
	if err != nil {
		return err
	}

	isZip := false
	for _, o := range s.outputs {
		if o.zipEntry != "" {
			isZip = true
			break
		}
	}

	if !isZip {
		o := s.outputs[0]
		return write(o, body)
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("%s: %w", s.url, err)
	}
	for _, o := range s.outputs {
		data, err := readZipEntry(zr, o.zipEntry)
		if err != nil {
			return fmt.Errorf("%s: %w", s.url, err)
		}
		if err := write(o, data); err != nil {
			return err
		}
	}
	return nil
}

func write(o output, data []byte) error {
	if got := sha256hex(data); got != o.sha256 {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s (update the manifest if this is an intentional bump)", o.dest, got, o.sha256)
	}
	if err := os.MkdirAll(filepath.Dir(o.dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(o.dest, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("genfonts: wrote %s (%d bytes)\n", o.dest, len(data))
	return nil
}

func download(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: HTTP %d", url, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func readZipEntry(zr *zip.Reader, name string) ([]byte, error) {
	for _, f := range zr.File {
		if f.Name == name {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			defer rc.Close()
			return io.ReadAll(rc)
		}
	}
	return nil, fmt.Errorf("zip entry %q not found", name)
}

func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sha256hex(b), nil
}

func sha256hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
