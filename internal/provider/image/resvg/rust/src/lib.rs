use std::slice;

use tiny_skia::{Pixmap, Transform};
use usvg::{Options, Tree};

/// Allocate `len` bytes in wasm linear memory and hand the pointer to the host.
/// Uses an exact-capacity Vec so `dealloc` is sound.
#[no_mangle]
pub extern "C" fn alloc(len: usize) -> *mut u8 {
    let mut buf: Vec<u8> = Vec::with_capacity(len);
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

/// Free a buffer previously returned by `alloc` (or by `render`). `len` MUST be
/// the length that was allocated (capacity == len, matching `alloc`).
#[no_mangle]
pub extern "C" fn dealloc(ptr: *mut u8, len: usize) {
    if ptr.is_null() || len == 0 {
        return;
    }
    unsafe {
        let _ = Vec::from_raw_parts(ptr, len, len);
    }
}

/// Render SVG -> PNG. Returns a pointer to the PNG bytes (length written to
/// *out_len_ptr), or null on failure. Fonts are a framed blob of repeated
/// [u32 LE len][bytes].
#[no_mangle]
pub extern "C" fn render(
    svg_ptr: *const u8,
    svg_len: usize,
    fonts_ptr: *const u8,
    fonts_len: usize,
    width: u32,
    height: u32,
    scale: f32,
    out_len_ptr: *mut usize,
) -> *mut u8 {
    unsafe {
        if !out_len_ptr.is_null() {
            *out_len_ptr = 0;
        }
    }

    if svg_ptr.is_null() || svg_len == 0 {
        return std::ptr::null_mut();
    }

    let svg: &[u8] = unsafe { slice::from_raw_parts(svg_ptr, svg_len) };

    let mut opt = Options::default();
    if !fonts_ptr.is_null() && fonts_len > 0 {
        let blob: &[u8] = unsafe { slice::from_raw_parts(fonts_ptr, fonts_len) };
        // Load fonts and pick a default family in a scope so the `db` borrow of
        // `opt` is released before we assign `opt.font_family`.
        let default_family = {
            let db = opt.fontdb_mut();
            let mut off = 0usize;
            while off + 4 <= blob.len() {
                let n =
                    u32::from_le_bytes([blob[off], blob[off + 1], blob[off + 2], blob[off + 3]])
                        as usize;
                off += 4;
                if n == 0 || off + n > blob.len() {
                    break;
                }
                db.load_font_data(blob[off..off + n].to_vec());
                off += n;
            }
            let fam = db
                .faces()
                .next()
                .and_then(|face| face.families.first().map(|(name, _)| name.clone()));
            // Map the CSS generic families to a loaded font so font-family
            // "sans-serif"/"serif"/"monospace" resolve. (The real provider loads
            // DejaVu Sans/Serif/Mono and maps each generic to the right one; here
            // in the shim we point them all at the default family.)
            if let Some(ref f) = fam {
                db.set_serif_family(f.clone());
                db.set_sans_serif_family(f.clone());
                db.set_monospace_family(f.clone());
                db.set_cursive_family(f.clone());
                db.set_fantasy_family(f.clone());
            }
            fam
        };
        if let Some(family) = default_family {
            opt.font_family = family;
        }
    }

    let tree: Tree = match Tree::from_data(svg, &opt) {
        Ok(t) => t,
        Err(_) => return std::ptr::null_mut(),
    };

    let size = tree.size();
    let (out_w, out_h, sx, sy) = if width > 0 && height > 0 {
        (
            width,
            height,
            width as f32 / size.width(),
            height as f32 / size.height(),
        )
    } else if width > 0 {
        // Width only: preserve aspect ratio.
        let s = width as f32 / size.width();
        (width, (size.height() * s).round().max(1.0) as u32, s, s)
    } else if height > 0 {
        // Height only: preserve aspect ratio.
        let s = height as f32 / size.height();
        ((size.width() * s).round().max(1.0) as u32, height, s, s)
    } else {
        let s = if scale > 0.0 { scale } else { 1.0 };
        (
            (size.width() * s).round().max(1.0) as u32,
            (size.height() * s).round().max(1.0) as u32,
            s,
            s,
        )
    };

    let mut pixmap = match Pixmap::new(out_w.max(1), out_h.max(1)) {
        Some(p) => p,
        None => return std::ptr::null_mut(),
    };

    resvg::render(&tree, Transform::from_scale(sx, sy), &mut pixmap.as_mut());

    let png = match pixmap.encode_png() {
        Ok(bytes) => bytes,
        Err(_) => return std::ptr::null_mut(),
    };

    let len = png.len();
    let out = alloc(len);
    if out.is_null() {
        return std::ptr::null_mut();
    }
    unsafe {
        std::ptr::copy_nonoverlapping(png.as_ptr(), out, len);
        if !out_len_ptr.is_null() {
            *out_len_ptr = len;
        }
    }
    out
}
