// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod command_security;
mod credentials;
mod path_security;
mod pty;
mod browser;
mod pack_fs;
mod ollama;

use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use std::hash::{Hash, Hasher};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use tauri::{Emitter, Manager, WebviewUrl, WebviewWindowBuilder};
use tauri_plugin_shell::{
    process::{CommandChild as ShellCommandChild, CommandEvent},
    ShellExt,
};

#[derive(Debug, Serialize, Deserialize)]
struct CommandResult {
    id: String,
    command: String,
    exit_code: i32,
    stdout: String,
    stderr: String,
    duration_ms: u64,
    success: bool,
}

// PTY commands live in pty.rs

/// Paths returned by native pack directory / zip pickers (outside Tauri fs allowlist).
pub(crate) type PackPathAllowlist = Arc<Mutex<HashSet<String>>>;

pub(crate) fn allowlist_insert(allowlist: &PackPathAllowlist, path: &str) -> Result<(), String> {
    let abs = path_security::canonicalize_for_allowlist(path)?;
    allowlist
        .lock()
        .unwrap()
        .insert(abs.to_string_lossy().into_owned());
    Ok(())
}

pub(crate) fn path_allowed(
    roots: &[String],
    allowlist: &PackPathAllowlist,
    candidate: &str,
) -> Result<PathBuf, String> {
    if let Ok(p) = path_security::within_any_root(roots, candidate) {
        return Ok(p);
    }
    let guard = allowlist.lock().unwrap();
    let entries: Vec<String> = guard.iter().cloned().collect();
    drop(guard);
    path_security::within_any_allowlist(&entries, candidate)
}

// ── One-off command execution (used for suggestion approve) ─────────────

#[tauri::command]
async fn execute_command(
    command: String,
    working_dir: Option<String>,
    allowed_roots: Vec<String>,
    user_approved: Option<bool>,
    app_handle: tauri::AppHandle,
) -> Result<CommandResult, String> {
    let approved = user_approved.unwrap_or(false);
    if !command_security::command_allowed_with_approval(&command, approved) {
        return Err(format!(
            "command not allowed: {} (approve only allowlisted commands)",
            command.lines().next().unwrap_or(&command)
        ));
    }
    if allowed_roots.is_empty() {
        return Err("no workspace roots configured".into());
    }
    path_security::validate_working_dir(&allowed_roots, working_dir.as_deref())?;

    let start_time = Instant::now();
    let command_id = uuid::Uuid::new_v4().to_string();

    let parts: Vec<&str> = if cfg!(target_os = "windows") {
        vec!["cmd", "/c", &command]
    } else {
        vec!["sh", "-c", &command]
    };

    let mut cmd = Command::new(parts[0]);
    cmd.args(&parts[1..]);
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::piped());

    if let Some(dir) = working_dir {
        cmd.current_dir(dir);
    }

    let output = cmd
        .output()
        .map_err(|e| format!("Failed to execute command: {}", e))?;

    let duration = start_time.elapsed();
    let stdout = String::from_utf8_lossy(&output.stdout).to_string();
    let stderr = String::from_utf8_lossy(&output.stderr).to_string();

    let result = CommandResult {
        id: command_id,
        command,
        exit_code: output.status.code().unwrap_or(-1),
        stdout,
        stderr,
        duration_ms: duration.as_millis() as u64,
        success: output.status.success(),
    };

    app_handle
        .emit("command-executed", &result)
        .map_err(|e| format!("Failed to emit event: {}", e))?;

    Ok(result)
}

// browser commands live in browser.rs

const MAX_ATTACH_BYTES: usize = 80_000;
const MAX_ATTACH_COUNT: usize = 12;
const MAX_ATTACH_TOTAL: usize = 350_000;

fn is_binary_attachment_path(path: &str) -> bool {
    let ext = std::path::Path::new(path)
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_ascii_lowercase();
    matches!(
        ext.as_str(),
        "png"
            | "jpg"
            | "jpeg"
            | "gif"
            | "webp"
            | "ico"
            | "svg"
            | "bmp"
            | "zip"
            | "tar"
            | "gz"
            | "pdf"
            | "mp4"
            | "mp3"
            | "wav"
            | "exe"
            | "dll"
            | "so"
            | "dylib"
            | "woff"
            | "woff2"
            | "ttf"
            | "eot"
            | "gguf"
            | "bin"
    )
}

fn infer_language_from_path(path: &str) -> String {
    let ext = std::path::Path::new(path)
        .extension()
        .and_then(|e| e.to_str())
        .unwrap_or("")
        .to_ascii_lowercase();
    let lang = match ext.as_str() {
        "go" => "go",
        "rs" => "rust",
        "py" => "python",
        "ts" => "typescript",
        "tsx" => "tsx",
        "js" => "javascript",
        "jsx" => "jsx",
        "md" => "markdown",
        "json" => "json",
        "yaml" | "yml" => "yaml",
        "toml" => "toml",
        "sql" => "sql",
        "sh" => "bash",
        "tf" | "hcl" => "hcl",
        "html" => "html",
        "css" => "css",
        "scss" => "scss",
        "vue" => "vue",
        "rb" => "ruby",
        "java" => "java",
        "kt" => "kotlin",
        "swift" => "swift",
        "c" | "h" => "c",
        "cpp" | "cc" | "cxx" => "cpp",
        "cs" => "csharp",
        _ => "text",
    };
    lang.to_string()
}

#[derive(Debug, Serialize, Deserialize)]
struct PromptAttachmentRead {
    path: String,
    language: String,
    content: String,
}


/// Register an absolute path from a native file/folder picker (pack dev / custom install).
/// Path is canonicalized; `..` escapes are rejected.
// pack_fs commands live in pack_fs.rs

#[tauri::command]
async fn read_prompt_attachment_paths(
    paths: Vec<String>,
    allowed_roots: Vec<String>,
) -> Result<Vec<PromptAttachmentRead>, String> {
    let mut out = Vec::new();
    let mut total = 0usize;
    for path in paths {
        if out.len() >= MAX_ATTACH_COUNT {
            break;
        }
        let path = path.trim();
        if path.is_empty() || is_binary_attachment_path(path) {
            continue;
        }
        path_security::within_any_root(&allowed_roots, path)?;
        let meta = std::fs::metadata(path).map_err(|e| format!("{}: {}", path, e))?;
        if !meta.is_file() {
            continue;
        }
        let raw = std::fs::read_to_string(path).map_err(|e| format!("{}: {}", path, e))?;
        let mut content = raw;
        if content.len() > MAX_ATTACH_BYTES {
            content.truncate(MAX_ATTACH_BYTES);
            content.push_str("\n[truncated client-side]");
        }
        if total + content.len() > MAX_ATTACH_TOTAL {
            break;
        }
        total += content.len();
        out.push(PromptAttachmentRead {
            path: path.to_string(),
            language: infer_language_from_path(path),
            content,
        });
    }
    Ok(out)
}

#[derive(Debug, Serialize, Deserialize)]
struct DecodedWellImage {
    mime: String,
    content_base64: String,
}

fn decoding_result_to_samples(result: tiff::decoder::DecodingResult) -> Vec<f64> {
    match result {
        tiff::decoder::DecodingResult::U8(v) => v.into_iter().map(|x| x as f64 / 255.0).collect(),
        tiff::decoder::DecodingResult::U16(v) => {
            v.into_iter().map(|x| x as f64 / 65535.0).collect()
        }
        tiff::decoder::DecodingResult::U32(v) => {
            let max = v.iter().copied().max().unwrap_or(1) as f64;
            let denom = if max > 0.0 { max } else { 1.0 };
            v.into_iter().map(|x| x as f64 / denom).collect()
        }
        tiff::decoder::DecodingResult::I8(v) => v
            .into_iter()
            .map(|x| (x as i32 - i8::MIN as i32) as f64 / 255.0)
            .collect(),
        tiff::decoder::DecodingResult::I16(v) => v
            .into_iter()
            .map(|x| (x as i32 - i16::MIN as i32) as f64 / 65535.0)
            .collect(),
        tiff::decoder::DecodingResult::I32(v) => {
            let max = v.iter().map(|&x| x as i64).max().unwrap_or(1) as f64;
            let denom = if max > 0.0 { max } else { 1.0 };
            v.into_iter().map(|x| x as f64 / denom).collect()
        }
        tiff::decoder::DecodingResult::U64(v) => {
            let max = v.iter().copied().max().unwrap_or(1) as f64;
            let denom = if max > 0.0 { max } else { 1.0 };
            v.into_iter().map(|x| x as f64 / denom).collect()
        }
        tiff::decoder::DecodingResult::I64(v) => {
            let max = v.iter().map(|&x| x as i128).max().unwrap_or(1) as f64;
            let denom = if max > 0.0 { max } else { 1.0 };
            v.into_iter().map(|x| x as f64 / denom).collect()
        }
        tiff::decoder::DecodingResult::F32(v) => v.into_iter().map(|x| x as f64).collect(),
        tiff::decoder::DecodingResult::F64(v) => v,
    }
}

#[tauri::command]
fn decode_scan_well_tiff(
    absolute_path: String,
    allowed_roots: Vec<String>,
) -> Result<DecodedWellImage, String> {
    use image::ImageEncoder;

    path_security::within_any_root(&allowed_roots, &absolute_path)?;

    let data =
        std::fs::read(&absolute_path).map_err(|e| format!("read {}: {}", absolute_path, e))?;
    let mut decoder = tiff::decoder::Decoder::new(std::io::Cursor::new(&data))
        .map_err(|e| format!("TIFF decoder: {}", e))?;
    let (w, h) = decoder
        .dimensions()
        .map_err(|e| format!("TIFF dimensions: {}", e))?;
    let decoded = decoder
        .read_image()
        .map_err(|e| format!("decode TIFF: {}", e))?;
    let samples = decoding_result_to_samples(decoded);
    let expected = (w as usize) * (h as usize);
    if samples.len() < expected {
        return Err(format!("TIFF sample count {} < {}x{}", samples.len(), w, h));
    }
    let mut min_v = f64::MAX;
    let mut max_v = f64::MIN;
    for &v in &samples[..expected] {
        if v < min_v {
            min_v = v;
        }
        if v > max_v {
            max_v = v;
        }
    }
    if max_v <= min_v {
        max_v = min_v + 1.0;
    }
    let scale = 255.0 / (max_v - min_v);
    let mut gray = image::GrayImage::new(w, h);
    for y in 0..h {
        for x in 0..w {
            let idx = (y as usize) * (w as usize) + (x as usize);
            let v = samples[idx];
            let n = ((v - min_v) * scale).round().clamp(0.0, 255.0) as u8;
            gray.put_pixel(x, y, image::Luma([n]));
        }
    }
    let mut png_buf = Vec::new();
    image::codecs::png::PngEncoder::new(&mut png_buf)
        .write_image(gray.as_raw(), w, h, image::ExtendedColorType::L8)
        .map_err(|e| format!("encode PNG: {}", e))?;
    Ok(DecodedWellImage {
        mime: "image/png".to_string(),
        content_base64: base64::Engine::encode(
            &base64::engine::general_purpose::STANDARD,
            &png_buf,
        ),
    })
}

#[tauri::command]
async fn open_markdown_preview(
    workspace_id: String,
    file_path: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Create unique window ID based on file path hash
    let mut hasher = std::collections::hash_map::DefaultHasher::new();
    file_path.hash(&mut hasher);
    let window_id = format!("md-preview-{:x}", hasher.finish());

    // Extract filename for window title
    let filename = std::path::Path::new(&file_path)
        .file_name()
        .and_then(|name| name.to_str())
        .unwrap_or("Markdown Preview");

    // Check if window already exists
    if app_handle.get_webview_window(window_id.as_str()).is_some() {
        // Focus existing window
        if let Some(window) = app_handle.get_webview_window(window_id.as_str()) {
            let _ = window.set_focus();
        }
        return Ok(());
    }

    // Create new window
    let window = WebviewWindowBuilder::new(
        &app_handle,
        window_id.as_str(),
        WebviewUrl::App(
            format!(
                "?preview=true&workspace={}&path={}",
                urlencoding::encode(&workspace_id),
                urlencoding::encode(&file_path)
            )
            .into(),
        ),
    )
    .title(format!("{} - Markdown Preview", filename))
    .inner_size(800.0, 600.0)
    .min_inner_size(400.0, 300.0)
    .resizable(true)
    .center()
    .build()
    .map_err(|e| format!("Failed to create window: {}", e))?;

    let _ = window.set_focus();
    Ok(())
}

// ── Sidecar lifecycle ───────────────────────────────────────────────

use std::sync::atomic::{AtomicBool, Ordering};

type SidecarChild = Arc<Mutex<Option<ShellCommandChild>>>;

static SIDECAR_READY: AtomicBool = AtomicBool::new(false);
static SHUTTING_DOWN: AtomicBool = AtomicBool::new(false);

pub(crate) fn app_is_shutting_down() -> bool {
    SHUTTING_DOWN.load(Ordering::SeqCst)
}

// ollama helpers live in ollama.rs

/// Default packaged hub listen port (matches desktop hubUrl DEFAULT_HUB_HTTP).
pub(crate) const DEFAULT_HUB_PORT: u16 = 18765;

/// Port for the hub this app manages (from NEURAL_JUNKIE_HUB_URL / VITE_NJ_HUB_URL, else 18765).
fn managed_hub_port() -> u16 {
    let base = std::env::var("NEURAL_JUNKIE_HUB_URL")
        .or_else(|_| std::env::var("VITE_NJ_HUB_URL"))
        .unwrap_or_else(|_| format!("http://127.0.0.1:{DEFAULT_HUB_PORT}"));
    parse_hub_port(&base).unwrap_or(DEFAULT_HUB_PORT)
}

fn parse_hub_port(base: &str) -> Option<u16> {
    let trimmed = base.trim().trim_end_matches('/');
    let without_scheme = trimmed
        .strip_prefix("http://")
        .or_else(|| trimmed.strip_prefix("https://"))
        .unwrap_or(trimmed);
    let host_port = without_scheme.split('/').next().unwrap_or(without_scheme);
    // IPv6 in brackets: [::1]:18765 — uncommon for local hub; handle host:port.
    if let Some((host, port_str)) = host_port.rsplit_once(':') {
        if !host.is_empty() && !host.contains(']') {
            return port_str.parse().ok();
        }
    }
    None
}

fn wait_for_hub_stopped(timeout: std::time::Duration) -> bool {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if !hub_health_once() {
            return true;
        }
        std::thread::sleep(std::time::Duration::from_millis(200));
    }
    !hub_health_once()
}

fn hub_health_once() -> bool {
    let health_url = dev_hub_health_url();
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_millis(800))
        .build();
    let Ok(client) = client else {
        return false;
    };
    match client.get(&health_url).send() {
        Ok(resp) if resp.status().is_success() => resp
            .json::<serde_json::Value>()
            .ok()
            .and_then(|v| v.get("status").and_then(|s| s.as_str()).map(|s| s == "ok"))
            .unwrap_or(false),
        _ => false,
    }
}

/// Reap orphan/external hubs on the managed port (packaged builds only).
fn stop_managed_hub_on_port() {
    let port = managed_hub_port();
    eprintln!("Stopping hub listeners on port {port}…");
    ollama::kill_listeners_on_port(port);
    let _ = wait_for_hub_stopped(std::time::Duration::from_secs(8));
}

// ollama commands live in ollama.rs

fn dev_hub_health_url() -> String {
    let base = std::env::var("NEURAL_JUNKIE_HUB_URL")
        .or_else(|_| std::env::var("VITE_NJ_HUB_URL"))
        .unwrap_or_else(|_| "http://127.0.0.1:18765".to_string());
    let base = base.trim_end_matches('/');
    format!("{}/api/health", base)
}

fn wait_for_server_health(timeout: std::time::Duration) -> bool {
    let health_url = dev_hub_health_url();
    let start = Instant::now();
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(2))
        .build()
        .unwrap();
    while start.elapsed() < timeout {
        if let Ok(resp) = client.get(&health_url).send() {
            if resp.status().is_success() {
                if let Ok(v) = resp.json::<serde_json::Value>() {
                    if v.get("status").and_then(|s| s.as_str()) == Some("ok") {
                        return true;
                    }
                }
            }
        }
        std::thread::sleep(std::time::Duration::from_millis(300));
    }
    false
}

fn spawn_sidecar(
    app: &tauri::AppHandle,
    bundled_ollama: Option<&PathBuf>,
) -> Result<ShellCommandChild, String> {
    let mut envs = HashMap::from([("OLLAMA_HOST".to_string(), "127.0.0.1:11434".to_string())]);
    // Debug/dev sidecars keep loopback parity with `make server`. Release builds
    // require a real desktop session (X-NJ-Session) for mutations.
    if cfg!(debug_assertions) {
        envs.insert(
            "NEURAL_JUNKIE_RELAXED_LOCAL".to_string(),
            "1".to_string(),
        );
    }
    if let Some(models_dir) = ollama::ollama_models_dir(app).to_str() {
        envs.insert("OLLAMA_MODELS".to_string(), models_dir.to_string());
    }
    if let Some(bin) = bundled_ollama {
        if let Some(path) = bin.to_str() {
            envs.insert("NJ_BUNDLED_OLLAMA".to_string(), path.to_string());
        }
    }

    let (mut rx, child) = app
        .shell()
        .sidecar("nj-server")
        .map_err(|e| format!("Failed to create sidecar command: {}", e))?
        .envs(envs)
        .spawn()
        .map_err(|e| format!("Failed to spawn sidecar: {}", e))?;

    // Drain sidecar stdout/stderr in background so pipe buffers don't fill
    std::thread::spawn(move || {
        while let Some(event) = rx.blocking_recv() {
            match event {
                CommandEvent::Stdout(line) => {
                    eprintln!("[nj-server] {}", String::from_utf8_lossy(&line))
                }
                CommandEvent::Stderr(line) => {
                    eprintln!("[nj-server err] {}", String::from_utf8_lossy(&line))
                }
                CommandEvent::Terminated(_) => break,
                _ => {}
            }
        }
    });

    Ok(child)
}

#[tauri::command]
async fn get_server_status() -> Result<bool, String> {
    Ok(SIDECAR_READY.load(Ordering::Relaxed))
}



fn shutdown_managed_processes(
    pty_sessions: &pty::PtySessions,
    sidecar_state: &SidecarChild,
    ollama_state: &ollama::OllamaChild,
) {
    if !begin_shutdown(&SHUTTING_DOWN) {
        return;
    }

    // Soft-unload models NJ loaded before killing the hub (which may be SIGKILL'd).
    ollama::unload_session_ollama_models_on_exit();

    let sessions = {
        let mut guard = pty_sessions.lock().unwrap();
        guard
            .drain()
            .map(|(_, session)| session)
            .collect::<Vec<_>>()
    };
    for mut session in sessions {
        let _ = session._child.kill();
        let _ = session._child.wait();
    }

    if let Some(child) = sidecar_state.lock().unwrap().take() {
        let _ = child.kill();
    }
    // Packaged builds own the hub on the managed port. Reap orphans the same way we
    // clear Ollama on 11434 — tracked-child kill alone can miss an external/stale hub.
    if !cfg!(debug_assertions) {
        stop_managed_hub_on_port();
    }
    ollama::stop_bundled_ollama_child(ollama_state);
    SIDECAR_READY.store(false, Ordering::SeqCst);
}

fn begin_shutdown(flag: &AtomicBool) -> bool {
    !flag.swap(true, Ordering::SeqCst)
}

#[tauri::command]
fn prepare_for_update(app: tauri::AppHandle) -> Result<(), String> {
    let pty_sessions = app.state::<pty::PtySessions>().inner().clone();
    let sidecar_state = app.state::<SidecarChild>().inner().clone();
    let ollama_state = app.state::<ollama::OllamaChild>().inner().clone();
    shutdown_managed_processes(&pty_sessions, &sidecar_state, &ollama_state);
    Ok(())
}

fn bootstrap_token_path() -> Result<PathBuf, String> {
    Ok(dirs::home_dir()
        .ok_or_else(|| "home directory not found".to_string())?
        .join(".neural-junkie")
        .join("bootstrap.token"))
}

fn load_bootstrap_token() -> Result<String, String> {
    let path = bootstrap_token_path()?;
    let raw = std::fs::read_to_string(&path)
        .map_err(|e| format!("could not read {}: {}", path.display(), e))?;
    let token = raw.trim().to_string();
    if token.is_empty() {
        return Err("Bootstrap token file is empty".into());
    }
    Ok(token)
}

/// Debug-only: read bootstrap token. Release builds must use mint_hub_admin_session.
#[tauri::command]
fn read_hub_bootstrap_token() -> Result<String, String> {
    if cfg!(not(debug_assertions)) {
        return Err(
            "bootstrap token is not readable in release builds; use Unlock with local mint"
                .into(),
        );
    }
    load_bootstrap_token()
}

/// Mint an admin hub session without exposing the bootstrap secret to the webview.
#[tauri::command]
fn mint_hub_admin_session(username: String, hub_url: String) -> Result<serde_json::Value, String> {
    let token = load_bootstrap_token()?;
    let base = hub_url.trim().trim_end_matches('/');
    if !(base.starts_with("http://127.0.0.1")
        || base.starts_with("http://localhost")
        || base.starts_with("https://127.0.0.1")
        || base.starts_with("https://localhost"))
    {
        return Err("admin mint only allowed against loopback hub URL".into());
    }
    let url = format!("{}/api/auth/session", base);
    let who = {
        let t = username.trim();
        if t.is_empty() {
            "local"
        } else {
            t
        }
    };
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(10))
        .build()
        .map_err(|e| e.to_string())?;
    let resp = client
        .post(&url)
        .header("Content-Type", "application/json")
        .header("X-NJ-Bootstrap", &token)
        .json(&serde_json::json!({ "username": who, "role": "admin" }))
        .send()
        .map_err(|e| format!("admin mint request failed: {}", e))?;
    if !resp.status().is_success() {
        let status = resp.status();
        let body = resp.text().unwrap_or_default();
        return Err(format!(
            "admin mint failed: {} {}",
            status,
            body.trim()
        ));
    }
    resp.json::<serde_json::Value>()
        .map_err(|e| format!("admin mint decode: {}", e))
}

/// Open WebView DevTools (packaged builds block right-click unless Option+click).
#[tauri::command]
fn open_devtools(app: tauri::AppHandle) -> Result<(), String> {
    app.get_webview_window("main")
        .ok_or_else(|| "main window not found".to_string())?
        .open_devtools();
    Ok(())
}

/// Probe hub WebSocket upgrade from native code (bypasses WebView CSP / mixed-content).
#[tauri::command]
fn probe_hub_websocket(hub_url: String, channel: String) -> Result<String, String> {
    let base = hub_url.trim().trim_end_matches('/');
    let ws_base = base
        .replacen("https://", "wss://", 1)
        .replacen("http://", "ws://", 1);
    let ch = channel.trim();
    let ch = if ch.is_empty() { "general" } else { ch };
    let ws_url = format!("{}/ws?channel={}", ws_base, ch);

    let output = std::process::Command::new("curl")
        .args([
            "-s",
            "-o",
            "/dev/null",
            "-w",
            "%{http_code}",
            "--max-time",
            "3",
            "-H",
            "Connection: Upgrade",
            "-H",
            "Upgrade: websocket",
            "-H",
            "Sec-WebSocket-Version: 13",
            "-H",
            "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==",
            &ws_url,
        ])
        .output()
        .map_err(|e| format!("curl failed: {}", e))?;

    if !output.status.success() && output.status.code() != Some(22) {
        let stderr = String::from_utf8_lossy(&output.stderr);
        return Err(format!("curl exit {:?}: {}", output.status.code(), stderr.trim()));
    }

    let code = String::from_utf8_lossy(&output.stdout).trim().to_string();
    if code == "101" {
        Ok(format!("OK (HTTP 101) {}", ws_url))
    } else {
        Ok(format!("HTTP {} (expected 101) {}", code, ws_url))
    }
}

fn main() {
    let pty_sessions: pty::PtySessions = Arc::new(Mutex::new(HashMap::new()));
    let sidecar_state: SidecarChild = Arc::new(Mutex::new(None));
    let ollama_state: ollama::OllamaChild = Arc::new(Mutex::new(None));
    let cleanup_ptys = pty_sessions.clone();
    let cleanup_sidecar = sidecar_state.clone();
    let cleanup_ollama = ollama_state.clone();

    let app = tauri::Builder::default()
        .plugin(tauri_plugin_dialog::init())
        .plugin(tauri_plugin_fs::init())
        .plugin(tauri_plugin_opener::init())
        .plugin(tauri_plugin_os::init())
        .plugin(tauri_plugin_process::init())
        .plugin(tauri_plugin_shell::init())
        .plugin(tauri_plugin_store::Builder::default().build())
        .plugin(tauri_plugin_updater::Builder::new().build())
        .manage(pty_sessions)
        .manage(Arc::new(Mutex::new(HashSet::<String>::new())) as PackPathAllowlist)
        .manage(sidecar_state)
        .manage(ollama_state)
        .setup(|app| {
            let sidecar_state = app.state::<SidecarChild>().inner().clone();
            let ollama_state = app.state::<ollama::OllamaChild>().inner().clone();
            let app_handle = app.handle().clone();

            // Kill any orphan hub on the managed port before the UI can connect to it.
            if !cfg!(debug_assertions) {
                stop_managed_hub_on_port();
            }

            std::thread::spawn(move || {
                // Only spawn sidecar in production builds; in dev the server
                // is started separately via `make server` or `make refresh`.
                if cfg!(debug_assertions) {
                    // Dev: start bundled Ollama when fetch-ollama artifacts exist (parity with production).
                    if !ollama::is_ollama_healthy() {
                        if let Some(bin) = ollama::spawn_bundled_ollama(&app_handle, &ollama_state) {
                            eprintln!("Dev: started bundled Ollama at {}", bin.display());
                        } else {
                            eprintln!(
                                "Dev: Ollama not running. Use `make start-all` (auto-starts) or `ollama serve`."
                            );
                        }
                    }

                    // Poll for an already-running hub (longer window: first Rust build + hub start).
                    if wait_for_server_health(std::time::Duration::from_secs(120)) {
                        SIDECAR_READY.store(true, Ordering::Relaxed);
                        let _ = app_handle.emit("server-ready", true);
                    } else {
                        let msg = format!(
                            "Neural Junkie hub not healthy at {}. From neural-junkie run: make server. If port 18765 is in use, set NEURAL_JUNKIE_HUB_URL and VITE_NJ_HUB_URL to match (e.g. http://127.0.0.1:18766) and start the hub with -addr :18766.",
                            dev_hub_health_url()
                        );
                        let _ = app_handle.emit("server-error", msg);
                    }
                    return;
                }

                let bundled_ollama = if ollama::is_ollama_healthy() {
                    ollama::bundled_ollama_runtime_dir(&app_handle)
                        .map(|dir| ollama::bundled_ollama_binary(&dir))
                        .filter(|bin| bin.exists())
                } else {
                    ollama::spawn_bundled_ollama(&app_handle, &ollama_state)
                };

                // Clear stale hubs on the managed port before spawn so health checks
                // don't pass against an orphan from a previous session / make run-hub.
                stop_managed_hub_on_port();

                match spawn_sidecar(&app_handle, bundled_ollama.as_ref()) {
                    Ok(child) => {
                        *sidecar_state.lock().unwrap() = Some(child);

                        if wait_for_server_health(std::time::Duration::from_secs(30)) {
                            SIDECAR_READY.store(true, Ordering::Relaxed);
                            let _ = app_handle.emit("server-ready", true);
                        } else {
                            let _ = app_handle.emit("server-error", "Server started but health check timed out");
                        }
                    }
                    Err(e) => {
                        eprintln!("Failed to start sidecar: {}", e);
                        let _ = app_handle.emit("server-error", e);
                    }
                }
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            pack_fs::register_pack_path,
            pack_fs::pick_pack_directory,
            pack_fs::write_pack_scaffold,
            pack_fs::read_pack_yaml_from_dir,
            pack_fs::write_pack_yaml_to_dir,
            pack_fs::zip_pack_directory,
            pack_fs::read_pack_zip_base64,
            read_prompt_attachment_paths,
            execute_command,
            pty::create_pty_session,
            pty::get_pty_session_status,
            pty::write_pty_session,
            pty::resize_pty_session,
            pty::close_pty_session,
            decode_scan_well_tiff,
            open_markdown_preview,
            browser::open_browser_window,
            browser::close_browser_window,
            browser::capture_browser_screenshot,
            browser::navigate_browser,
            browser::create_embedded_browser,
            browser::update_browser_position,
            browser::navigate_embedded_browser,
            browser::destroy_embedded_browser,
            get_server_status,
            ollama::get_ollama_runtime_status,
            ollama::start_bundled_ollama,
            ollama::stop_bundled_ollama,
            ollama::restart_bundled_ollama,
            prepare_for_update,
            credentials::encrypt_credential_blob,
            credentials::decrypt_credential_blob,
            read_hub_bootstrap_token,
            mint_hub_admin_session,
            open_devtools,
            probe_hub_websocket
        ])
        .build(tauri::generate_context!())
        .expect("error while running tauri application");

    app.run(move |_app_handle, event| {
        if let tauri::RunEvent::Exit = event {
            shutdown_managed_processes(&cleanup_ptys, &cleanup_sidecar, &cleanup_ollama);
        }
    });
}

#[cfg(test)]
mod lifecycle_tests {
    use super::{begin_shutdown, parse_hub_port, DEFAULT_HUB_PORT};
    use std::sync::atomic::AtomicBool;

    #[test]
    fn shutdown_is_idempotent() {
        let flag = AtomicBool::new(false);
        assert!(begin_shutdown(&flag));
        assert!(!begin_shutdown(&flag));
    }

    #[test]
    fn parse_hub_port_from_url() {
        assert_eq!(parse_hub_port("http://127.0.0.1:18765"), Some(18765));
        assert_eq!(parse_hub_port("http://127.0.0.1:18766/"), Some(18766));
        assert_eq!(parse_hub_port("https://localhost:19999/api"), Some(19999));
        assert_eq!(parse_hub_port("http://127.0.0.1"), None);
        assert_eq!(DEFAULT_HUB_PORT, 18765);
    }
}
