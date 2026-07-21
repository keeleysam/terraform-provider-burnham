package typst

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/keeleysam/terraform-burnham/internal/provider/image"
)

// A document that exercises structured sys.inputs and the bundled fonts, rendered to every format.
const smokeSource = `#set text(font: "Noto Sans")
#set page(width: 200pt, height: auto, margin: 12pt)
= Report for #sys.inputs.customer.name
Level #sys.inputs.customer.level.`

func smokeReq(op string) Request {
	return Request{
		Op:     op,
		Source: smokeSource,
		Inputs: map[string]any{
			"customer": map[string]any{"name": "Ada", "level": 7},
		},
		Fonts: image.BundledFonts(),
	}
}

func TestRenderPDF(t *testing.T) {
	pages, err := Render(context.Background(), smokeReq("pdf"))
	if err != nil {
		t.Fatalf("render pdf: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("pdf pages = %d, want 1", len(pages))
	}
	if !bytes.HasPrefix(pages[0], []byte("%PDF-")) {
		t.Fatalf("pdf does not start with %%PDF-: %q", pages[0][:min(8, len(pages[0]))])
	}
}

func TestRenderPNG(t *testing.T) {
	pages, err := Render(context.Background(), Request{Op: "png", Source: smokeSource, Inputs: smokeReq("png").Inputs, Fonts: image.BundledFonts(), PPI: 96})
	if err != nil {
		t.Fatalf("render png: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("png pages = %d, want 1", len(pages))
	}
	// PNG magic number.
	if !bytes.HasPrefix(pages[0], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}) {
		t.Fatal("first page is not a PNG")
	}
}

func TestRenderSVGAndHTML(t *testing.T) {
	svg, err := Render(context.Background(), smokeReq("svg"))
	if err != nil {
		t.Fatalf("render svg: %v", err)
	}
	if len(svg) != 1 || !bytes.Contains(svg[0], []byte("<svg")) {
		t.Fatalf("svg output unexpected: %d pages", len(svg))
	}
	html, err := Render(context.Background(), smokeReq("html"))
	if err != nil {
		t.Fatalf("render html: %v", err)
	}
	if len(html) != 1 || !bytes.Contains(html[0], []byte("<!DOCTYPE html>")) {
		t.Fatal("html output unexpected")
	}
	// The structured input must have reached the document.
	if !bytes.Contains(html[0], []byte("Ada")) {
		t.Fatal("sys.inputs did not reach the rendered document")
	}
}

func TestRenderCompileErrorIsEngineError(t *testing.T) {
	_, err := Render(context.Background(), Request{Op: "pdf", Source: "#panic(\"boom\")", Fonts: image.BundledFonts()})
	if err == nil {
		t.Fatal("expected a compile error")
	}
	var ee *EngineError
	if !errors.As(err, &ee) {
		t.Fatalf("expected *EngineError, got %T: %v", err, err)
	}
}
