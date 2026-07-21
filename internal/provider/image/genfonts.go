//go:build ignore

// Command genfonts downloads the bundled fonts that are not available as Go
// modules (the Noto text families and the Noto Color Emoji build) into fonts/,
// verifying each against a pinned SHA-256. It is run via `go generate` (see
// fonts.go); the downloaded .ttf files are gitignored, so a fresh checkout must
// run `go generate ./...` before the image package will compile.
//
// To update a font: bump the pinned ref below (and/or the per-font path), run
// this, and paste the new sha256 it reports on a mismatch.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// Pins. Noto Color Emoji ships tagged releases; the Noto text families live in
// the google/fonts monorepo, pinned to a commit so the bytes are reproducible.
const (
	emojiRef = "v2.051"                                   // github.com/googlefonts/noto-emoji release tag
	fontsRef = "2f6daa88e1e71320a6fe71cc91ecbfc018928737" // github.com/google/fonts commit
)

type font struct {
	dest, url, sha256 string
}

var manifest = []font{
	{
		dest:   "fonts/Noto-COLRv1.ttf",
		url:    "https://github.com/googlefonts/noto-emoji/raw/" + emojiRef + "/fonts/Noto-COLRv1.ttf",
		sha256: "0ae57fe58645638523ba35f388d93739d292539a9acb84df5700c81b1e1a28d2",
	},
	{
		dest:   "fonts/NotoSans.ttf",
		url:    gfonts("notosans", "NotoSans%5Bwdth%2Cwght%5D.ttf"),
		sha256: "bfb7bb691513f12e734dc346c03a03f784912432d7e3fa8e56efcf906fe86b3d",
	},
	{
		dest:   "fonts/NotoSans-Italic.ttf",
		url:    gfonts("notosans", "NotoSans-Italic%5Bwdth%2Cwght%5D.ttf"),
		sha256: "58e6e0ebd1931b29a365aa2d3e2ee9a9e831a3af7cf3ad1462d4e72154f0b291",
	},
	{
		dest:   "fonts/NotoSerif.ttf",
		url:    gfonts("notoserif", "NotoSerif%5Bwdth%2Cwght%5D.ttf"),
		sha256: "4d8e6761424656867019081a1a01336f3cb086982682698714054fc33f782713",
	},
	{
		dest:   "fonts/NotoSerif-Italic.ttf",
		url:    gfonts("notoserif", "NotoSerif-Italic%5Bwdth%2Cwght%5D.ttf"),
		sha256: "e87acbc6c0efd0d9a20d6a8cbbda2b266c14be3a3a6f5af8ec9d7b2460570ad1",
	},
	{
		dest:   "fonts/NotoSansMono.ttf",
		url:    gfonts("notosansmono", "NotoSansMono%5Bwdth%2Cwght%5D.ttf"),
		sha256: "2cb2adb378a8f574213e23df697050b83c54c27df465a2015552740b2769a081",
	},
}

func gfonts(family, file string) string {
	return "https://raw.githubusercontent.com/google/fonts/" + fontsRef + "/ofl/" + family + "/" + file
}

func main() {
	for _, f := range manifest {
		if err := ensure(f); err != nil {
			fmt.Fprintf(os.Stderr, "genfonts: %s: %v\n", f.dest, err)
			os.Exit(1)
		}
	}
}

func ensure(f font) error {
	if sum, err := hashFile(f.dest); err == nil && sum == f.sha256 {
		return nil // already present and correct
	}
	if err := os.MkdirAll(filepath.Dir(f.dest), 0o755); err != nil {
		return err
	}
	resp, err := http.Get(f.url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: HTTP %d", f.url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(body)
	if got := hex.EncodeToString(sum[:]); got != f.sha256 {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s (update the manifest if this is an intentional bump)", f.url, got, f.sha256)
	}
	if err := os.WriteFile(f.dest, body, 0o644); err != nil {
		return err
	}
	fmt.Printf("genfonts: fetched %s (%d bytes)\n", f.dest, len(body))
	return nil
}

func hashFile(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
