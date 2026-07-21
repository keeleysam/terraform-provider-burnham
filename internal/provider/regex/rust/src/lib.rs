use std::slice;

use fancy_regex::Regex;
use serde_json::{json, Value};

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

fn as_str<'a>(ptr: *const u8, len: usize) -> Result<&'a str, String> {
    if ptr.is_null() || len == 0 {
        return Ok("");
    }
    let bytes = unsafe { slice::from_raw_parts(ptr, len) };
    // Return an error rather than coercing invalid UTF-8 to "": a silent "" would produce quietly
    // wrong results (an empty pattern matches everywhere, empty input never matches). Terraform
    // strings are UTF-8, so this only fires on a corrupt buffer, but a clear error beats a wrong answer.
    std::str::from_utf8(bytes).map_err(|e| format!("input is not valid UTF-8: {e}"))
}

/// run executes a regex operation and returns a JSON envelope: `{"v": <result>}`
/// on success or `{"e": "<message>"}` on error (invalid pattern, backtrack limit,
/// invalid utf-8 in a group, ...). The length is written to `out_len`.
///
/// op: 0 = is_match (bool), 1 = captures (object of numbered + named groups, or
/// null), 2 = find_all (array of full matches), 3 = replace_all (string;
/// replacement supports $1 / ${name}), 4 = split (array).
#[no_mangle]
pub extern "C" fn run(
    op: u32,
    pat_ptr: *const u8,
    pat_len: usize,
    inp_ptr: *const u8,
    inp_len: usize,
    rep_ptr: *const u8,
    rep_len: usize,
    out_len: *mut usize,
) -> *mut u8 {
    let result = (|| -> Result<Value, String> {
        let pattern = as_str(pat_ptr, pat_len)?;
        let input = as_str(inp_ptr, inp_len)?;
        let replacement = as_str(rep_ptr, rep_len)?;
        compute(op, pattern, input, replacement)
    })();

    let envelope = match result {
        Ok(v) => json!({ "v": v }),
        Err(e) => json!({ "e": e }),
    };
    let bytes = serde_json::to_vec(&envelope)
        .unwrap_or_else(|_| br#"{"e":"failed to serialize result"}"#.to_vec());

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

fn compute(op: u32, pattern: &str, input: &str, replacement: &str) -> Result<Value, String> {
    let re = Regex::new(pattern).map_err(|e| format!("invalid pattern: {e}"))?;
    match op {
        0 => Ok(json!(re.is_match(input).map_err(|e| e.to_string())?)),
        1 => match re.captures(input).map_err(|e| e.to_string())? {
            None => Ok(Value::Null),
            Some(caps) => {
                let mut m = serde_json::Map::new();
                for i in 0..caps.len() {
                    if let Some(g) = caps.get(i) {
                        m.insert(i.to_string(), json!(g.as_str()));
                    }
                }
                for name in re.capture_names().flatten() {
                    if let Some(g) = caps.name(name) {
                        m.insert(name.to_string(), json!(g.as_str()));
                    }
                }
                Ok(Value::Object(m))
            }
        },
        2 => {
            let mut out = Vec::new();
            for m in re.find_iter(input) {
                out.push(json!(m.map_err(|e| e.to_string())?.as_str()));
            }
            Ok(Value::Array(out))
        }
        /*
            try_replacen(input, 0, ...) is the fallible form of replace_all: limit 0 means "all
            matches". We must use the try_ variant because replace_all/replacen unwrap internally,
            so a runtime error (e.g. the backtrack limit tripping on a catastrophic pattern) would
            panic; with panic = "abort" that traps the whole wasm instance instead of returning an
            error. Propagating the Err here keeps op 3 at parity with the other ops, which all
            surface a clean {"e": ...} envelope.
        */
        3 => Ok(json!(re
            .try_replacen(input, 0, replacement)
            .map_err(|e| e.to_string())?
            .into_owned())),
        4 => {
            let mut out = Vec::new();
            let mut last = 0usize;
            for m in re.find_iter(input) {
                let m = m.map_err(|e| e.to_string())?;
                out.push(json!(&input[last..m.start()]));
                last = m.end();
            }
            out.push(json!(&input[last..]));
            Ok(Value::Array(out))
        }
        _ => Err(format!("unknown op {op}")),
    }
}
