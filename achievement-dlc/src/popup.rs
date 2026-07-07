//! Steam-style popup toast rendered in a small borderless always-on-top
//! window (minifb + fontdue). Falls back to an OS notification upstream if a
//! window or font is unavailable (e.g. headless machines).

use anyhow::{Context, Result};
use minifb::{Window, WindowOptions};
use std::time::{Duration, Instant};

use crate::Unlock;

const W: usize = 400;
const H: usize = 104;
const BG: u32 = 0xFF1B1E26;
const ACCENT: u32 = 0xFFD9A441; // default gold bar + heading
const FG: u32 = 0xFFF2F3F5;
const MUTED: u32 = 0xFF9AA3B2;
const SHOW_FOR: Duration = Duration::from_millis(4500);
const MARGIN_X: i32 = 24;
const MARGIN_Y: i32 = 64; // leaves room for a taskbar/dock

/// Accent color per achievement tier (matches the web achievement wall).
fn tier_accent(tier: &str) -> u32 {
    match tier.to_ascii_lowercase().as_str() {
        "bronze" => 0xFFB87333,
        "silver" => 0xFFC0C4CC,
        "gold" => 0xFFD9A441,
        "platinum" => 0xFF67D8E0,
        _ => ACCENT,
    }
}

pub fn show(u: &Unlock) -> Result<()> {
    let font = load_font()?;
    let accent = tier_accent(&u.tier);
    let mut buf = vec![BG; W * H];

    // tier-colored accent bar on the left
    for y in 0..H {
        for x in 0..4 {
            buf[y * W + x] = accent;
        }
    }
    draw_text(&mut buf, &font, "ACHIEVEMENT UNLOCKED", 20.0, 14.0, 13.0, accent);
    draw_text(&mut buf, &font, &u.name, 20.0, 36.0, 22.0, FG);
    let mut desc = u.description.clone();
    if u.points > 0 {
        desc = format!("{}  (+{} pts)", desc, u.points);
    }
    draw_text(&mut buf, &font, &truncate(&desc, 52), 20.0, 70.0, 14.0, MUTED);

    let mut win = Window::new(
        "ArcNode Achievement",
        W,
        H,
        WindowOptions {
            borderless: true,
            topmost: true,
            none: true,
            ..WindowOptions::default()
        },
    )
    .context("open popup window")?;
    let (px, py) = match screen_size() {
        Some((sw, sh)) => (
            (sw as i32 - W as i32 - MARGIN_X).max(0),
            (sh as i32 - H as i32 - MARGIN_Y).max(0),
        ),
        None => (60, 60),
    };
    win.set_position(px as isize, py as isize);

    let start = Instant::now();
    while win.is_open() && start.elapsed() < SHOW_FOR {
        win.update_with_buffer(&buf, W, H).context("draw popup")?;
        std::thread::sleep(Duration::from_millis(33));
    }
    Ok(())
}

/// Primary display size in pixels, used to anchor the toast bottom-right.
#[cfg(target_os = "windows")]
fn screen_size() -> Option<(usize, usize)> {
    use windows::Win32::UI::WindowsAndMessaging::{GetSystemMetrics, SM_CXSCREEN, SM_CYSCREEN};
    let w = unsafe { GetSystemMetrics(SM_CXSCREEN) };
    let h = unsafe { GetSystemMetrics(SM_CYSCREEN) };
    (w > 0 && h > 0).then_some((w as usize, h as usize))
}

#[cfg(target_os = "macos")]
fn screen_size() -> Option<(usize, usize)> {
    let out = std::process::Command::new("osascript")
        .args(["-e", "tell application \"Finder\" to get bounds of window of desktop"])
        .output()
        .ok()?;
    let s = String::from_utf8_lossy(&out.stdout);
    let parts: Vec<i64> = s
        .trim()
        .split(',')
        .filter_map(|p| p.trim().parse().ok())
        .collect();
    match parts.as_slice() {
        [_, _, w, h] if *w > 0 && *h > 0 => Some((*w as usize, *h as usize)),
        _ => None,
    }
}

#[cfg(all(unix, not(target_os = "macos")))]
fn screen_size() -> Option<(usize, usize)> {
    // xrandr prints: "Screen 0: ..., current 1920 x 1080, maximum ..."
    let out = std::process::Command::new("xrandr").arg("--current").output().ok()?;
    let s = String::from_utf8_lossy(&out.stdout);
    let rest = &s[s.find("current ")? + "current ".len()..];
    let dims = rest.split(',').next()?;
    let mut it = dims.split('x').filter_map(|p| p.trim().parse::<usize>().ok());
    match (it.next(), it.next()) {
        (Some(w), Some(h)) if w > 0 && h > 0 => Some((w, h)),
        _ => None,
    }
}

fn truncate(s: &str, max_chars: usize) -> String {
    if s.chars().count() <= max_chars {
        return s.to_string();
    }
    let cut: String = s.chars().take(max_chars.saturating_sub(1)).collect();
    format!("{}…", cut)
}

fn load_font() -> Result<fontdue::Font> {
    // CJK-capable fonts first so non-latin achievement names render.
    let candidates: &[&str] = if cfg!(target_os = "windows") {
        &[
            "C:\\Windows\\Fonts\\msyh.ttc",
            "C:\\Windows\\Fonts\\segoeui.ttf",
            "C:\\Windows\\Fonts\\arial.ttf",
        ]
    } else if cfg!(target_os = "macos") {
        &[
            "/System/Library/Fonts/PingFang.ttc",
            "/System/Library/Fonts/Helvetica.ttc",
            "/System/Library/Fonts/SFNS.ttf",
        ]
    } else {
        &[
            "/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
            "/usr/share/fonts/noto-cjk/NotoSansCJK-Regular.ttc",
            "/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
            "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
            "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
            "/usr/share/fonts/TTF/DejaVuSans.ttf",
            "/usr/share/fonts/noto/NotoSans-Regular.ttf",
        ]
    };
    for path in candidates {
        if let Ok(bytes) = std::fs::read(path) {
            if let Ok(font) = fontdue::Font::from_bytes(bytes, fontdue::FontSettings::default()) {
                return Ok(font);
            }
        }
    }
    anyhow::bail!("no usable system font found")
}

fn draw_text(buf: &mut [u32], font: &fontdue::Font, text: &str, x: f32, y: f32, size: f32, color: u32) {
    let mut pen_x = x;
    for ch in text.chars() {
        let (metrics, bitmap) = font.rasterize(ch, size);
        let gx = (pen_x + metrics.xmin as f32) as i32;
        let gy = (y + size - metrics.height as f32 - metrics.ymin as f32) as i32;
        for row in 0..metrics.height {
            for col in 0..metrics.width {
                let px = gx + col as i32;
                let py = gy + row as i32;
                if px < 0 || py < 0 || px >= W as i32 || py >= H as i32 {
                    continue;
                }
                let cov = bitmap[row * metrics.width + col] as u32;
                if cov == 0 {
                    continue;
                }
                let idx = py as usize * W + px as usize;
                buf[idx] = blend(buf[idx], color, cov);
            }
        }
        pen_x += metrics.advance_width;
    }
}

fn blend(bg: u32, fg: u32, cov: u32) -> u32 {
    let mix = |b: u32, f: u32| (b * (255 - cov) + f * cov) / 255;
    let r = mix((bg >> 16) & 0xFF, (fg >> 16) & 0xFF);
    let g = mix((bg >> 8) & 0xFF, (fg >> 8) & 0xFF);
    let b = mix(bg & 0xFF, fg & 0xFF);
    0xFF00_0000 | (r << 16) | (g << 8) | b
}
