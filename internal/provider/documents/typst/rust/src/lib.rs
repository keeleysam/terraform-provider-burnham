//! WebAssembly shim around the Typst typesetting engine.
//!
//! The host passes a single JSON request and gets back a single JSON response, both over the flat
//! alloc/dealloc/run linear-memory ABI shared by the other burnham engines.
//!
//! Request: {"op":"pdf"|"png"|"svg", "source":"<typst markup>", "inputs":{...}, "files":{"path":"<base64>"},
//!           "fonts":["<base64>", ...], "ppi":144.0}
//! Response: {"pages":["<base64>", ...]} on success (one element for pdf, one per page for png/svg),
//!           or {"error":"<message>"} on failure.
//!
//! `inputs` is converted from JSON into native Typst values and exposed to the document as
//! `sys.inputs`, so a document reads `sys.inputs.customer.name` directly with no decode step.

use std::slice;

use base64::engine::general_purpose::STANDARD as B64;
use base64::Engine as _;
use serde_json::Value as Json;
use typst::foundations::{Array, Bytes, Dict, Str, Value};
use typst_as_lib::TypstEngine;
use typst_html::HtmlDocument;
use typst_layout::PagedDocument;
use typst_utils::Scalar;

/// Allocate `len` bytes in wasm linear memory for the host. Paired with `dealloc`.
#[no_mangle]
pub extern "C" fn alloc(len: usize) -> *mut u8 {
    let mut buf: Vec<u8> = Vec::with_capacity(len);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

/// Free a buffer previously returned by `alloc` (or `run`). `len` must match.
#[no_mangle]
pub extern "C" fn dealloc(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(ptr, len, len);
    }
}

fn as_bytes<'a>(ptr: *const u8, len: usize) -> &'a [u8] {
    if ptr.is_null() || len == 0 {
        return &[];
    }
    unsafe { slice::from_raw_parts(ptr, len) }
}

/// run compiles the requested Typst document and returns the JSON envelope described above. The
/// length is written to `out_len`.
#[no_mangle]
pub extern "C" fn run(req_ptr: *const u8, req_len: usize, out_len: *mut usize) -> *mut u8 {
    let envelope = match compute(as_bytes(req_ptr, req_len)) {
        Ok(pages) => serde_json::json!({ "pages": pages }),
        Err(e) => serde_json::json!({ "error": e }),
    };
    let bytes = serde_json::to_vec(&envelope)
        .unwrap_or_else(|_| br#"{"error":"failed to serialize result"}"#.to_vec());

    let len = bytes.len();
    let out = alloc(len);
    unsafe {
        std::ptr::copy_nonoverlapping(bytes.as_ptr(), out, len);
        if !out_len.is_null() {
            *out_len = len;
        }
    }
    out
}

fn compute(req: &[u8]) -> Result<Vec<String>, String> {
    let req: Json = serde_json::from_slice(req).map_err(|e| format!("invalid request json: {e}"))?;
    let op = req.get("op").and_then(|v| v.as_str()).ok_or("missing op")?;
    let source = req
        .get("source")
        .and_then(|v| v.as_str())
        .ok_or("missing source")?
        .to_string();
    let ppi = req.get("ppi").and_then(|v| v.as_f64()).unwrap_or(144.0);

    // Fonts: base64-encoded font files supplied by the host (bundled + user).
    let mut fonts: Vec<Vec<u8>> = Vec::new();
    if let Some(arr) = req.get("fonts").and_then(|v| v.as_array()) {
        for f in arr {
            if let Some(s) = f.as_str() {
                fonts.push(B64.decode(s).map_err(|e| format!("font base64: {e}"))?);
            }
        }
    }

    // Files: a virtual filesystem for #import/#include (.typ, treated as source text) and #image or
    // data loaders (everything else, treated as raw bytes). Values are base64.
    let mut sources: Vec<(String, String)> = Vec::new();
    let mut binaries: Vec<(String, Bytes)> = Vec::new();
    if let Some(obj) = req.get("files").and_then(|v| v.as_object()) {
        for (path, val) in obj {
            let b64 = val.as_str().ok_or("file values must be base64 strings")?;
            let raw = B64
                .decode(b64)
                .map_err(|e| format!("file base64 for {path}: {e}"))?;
            if path.ends_with(".typ") {
                let text =
                    String::from_utf8(raw).map_err(|_| format!("{path} is not valid UTF-8"))?;
                sources.push((path.clone(), text));
            } else {
                binaries.push((path.clone(), Bytes::new(raw)));
            }
        }
    }

    // Inputs: JSON object -> native Typst Dict, exposed as sys.inputs.
    let inputs = match req.get("inputs") {
        Some(v @ Json::Object(_)) => match json_to_value(v) {
            Value::Dict(d) => d,
            _ => Dict::new(),
        },
        _ => Dict::new(),
    };

    let mut builder = TypstEngine::builder().main_file(source).fonts(fonts);
    if !sources.is_empty() {
        // IntoSource is implemented for (&str, String); the owned `sources` outlives this call.
        builder = builder
            .with_static_source_file_resolver(sources.iter().map(|(k, v)| (k.as_str(), v.clone())));
    }
    if !binaries.is_empty() {
        // IntoFileId is implemented for &str; Bytes clones are cheap (Arc-backed).
        builder = builder
            .with_static_file_resolver(binaries.iter().map(|(k, v)| (k.as_str(), v.clone())));
    }
    let engine = builder.build();

    // HTML is a separate compile target (HtmlDocument, gated behind Typst's experimental Feature::Html)
    // and reflows to a single self-contained document, so it does not go through the paged pipeline.
    if op == "html" {
        let doc: HtmlDocument = engine
            .compile_with_input(inputs)
            .output
            .map_err(|e| format!("{e:?}"))?;
        let html = typst_html::html(&doc, &typst_html::HtmlOptions::default())
            .map_err(|e| format!("html export: {e:?}"))?;
        return Ok(vec![B64.encode(html.into_bytes())]);
    }

    let doc: PagedDocument = engine
        .compile_with_input(inputs)
        .output
        .map_err(|e| format!("{e:?}"))?;
    let pages = doc.pages();

    match op {
        "pdf" => {
            let pdf = typst_pdf::pdf(&doc, &typst_pdf::PdfOptions::default())
                .map_err(|e| format!("pdf export: {e:?}"))?;
            Ok(vec![B64.encode(pdf)])
        }
        "png" => {
            let mut ro = typst_render::RenderOptions::default();
            ro.pixel_per_pt = Scalar::new(ppi / 72.0);
            let mut out = Vec::with_capacity(pages.len());
            for p in pages {
                let pm = typst_render::render(p, &ro);
                let png = pm.encode_png().map_err(|e| format!("png encode: {e}"))?;
                out.push(B64.encode(png));
            }
            Ok(out)
        }
        "svg" => {
            let opts = typst_svg::SvgOptions::default();
            Ok(pages
                .iter()
                .map(|p| B64.encode(typst_svg::svg(p, &opts).into_bytes()))
                .collect())
        }
        other => Err(format!("unknown op {other}")),
    }
}

/// Convert a serde_json value into a native Typst value, so structured Terraform data reaches the
/// document as real dicts/arrays/numbers rather than JSON strings needing an in-document decode.
fn json_to_value(j: &Json) -> Value {
    match j {
        Json::Null => Value::None,
        Json::Bool(b) => Value::Bool(*b),
        Json::Number(n) => match n.as_i64() {
            Some(i) => Value::Int(i),
            None => Value::Float(n.as_f64().unwrap_or(0.0)),
        },
        Json::String(s) => Value::Str(Str::from(s.as_str())),
        Json::Array(a) => Value::Array(a.iter().map(json_to_value).collect::<Array>()),
        Json::Object(o) => {
            let mut d = Dict::new();
            for (k, v) in o {
                d.insert(Str::from(k.as_str()), json_to_value(v));
            }
            Value::Dict(d)
        }
    }
}
