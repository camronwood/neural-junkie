// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

mod command_security;
mod path_security;

use portable_pty::{native_pty_system, CommandBuilder, PtySize};
use serde::{Deserialize, Serialize};
use std::collections::{HashMap, HashSet};
use std::hash::{Hash, Hasher};
use std::io::{Read, Write};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use tauri::path::BaseDirectory;
use tauri::{Emitter, Manager, WebviewUrl, WebviewWindowBuilder};
use tauri_plugin_dialog::DialogExt;
use tauri_plugin_opener::OpenerExt;
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

// ── PTY session management ──────────────────────────────────────────────

struct PtySession {
    writer: Box<dyn Write + Send>,
    _child: Box<dyn portable_pty::Child + Send>,
    pair: portable_pty::PtyPair,
    process_id: Option<u32>,
}

type PtySessions = Arc<Mutex<HashMap<String, PtySession>>>;

#[derive(Debug, Serialize)]
struct PtySessionStatus {
    foreground_work: bool,
}

/// Paths returned by native pack directory / zip pickers (outside Tauri fs allowlist).
type PackPathAllowlist = Arc<Mutex<HashSet<String>>>;

fn allowlist_insert(allowlist: &PackPathAllowlist, path: &str) {
    let trimmed = path.trim();
    if !trimmed.is_empty() {
        allowlist.lock().unwrap().insert(trimmed.to_string());
    }
}

fn path_allowed(
    roots: &[String],
    allowlist: &PackPathAllowlist,
    candidate: &str,
) -> Result<PathBuf, String> {
    if let Ok(p) = path_security::within_any_root(roots, candidate) {
        return Ok(p);
    }
    let cand = candidate.trim();
    let guard = allowlist.lock().unwrap();
    for allowed in guard.iter() {
        let base = allowed.trim();
        if cand == base
            || cand.starts_with(&format!("{}/", base))
            || cand.starts_with(&format!("{}\\", base))
        {
            return Ok(PathBuf::from(cand));
        }
    }
    Err(format!("path not allowed: {}", candidate))
}

fn default_home() -> PathBuf {
    dirs::home_dir().unwrap_or_else(|| PathBuf::from("/"))
}

fn default_shell() -> String {
    std::env::var("SHELL").unwrap_or_else(|_| {
        if cfg!(target_os = "windows") {
            "cmd.exe".to_string()
        } else {
            "/bin/zsh".to_string()
        }
    })
}

#[tauri::command]
async fn create_pty_session(
    id: String,
    cwd: Option<String>,
    cols: Option<u16>,
    rows: Option<u16>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if SHUTTING_DOWN.load(Ordering::SeqCst) {
        return Err("application is shutting down".into());
    }
    let pty_system = native_pty_system();

    let size = PtySize {
        rows: rows.unwrap_or(24),
        cols: cols.unwrap_or(80),
        pixel_width: 0,
        pixel_height: 0,
    };

    let pair = pty_system
        .openpty(size)
        .map_err(|e| format!("Failed to open PTY: {}", e))?;

    let working_dir = match cwd.as_deref() {
        None | Some("~") | Some("") => default_home(),
        Some(p) if p.starts_with("~/") => default_home().join(&p[2..]),
        Some(p) => PathBuf::from(p),
    };

    let shell = default_shell();
    let mut cmd = CommandBuilder::new(&shell);
    // Launch as interactive login shell with a real terminal type so
    // backspace, clear, colors, and zsh/readline editing work.
    cmd.arg("-l");
    cmd.env("TERM", "xterm-256color");
    cmd.env("COLORTERM", "truecolor");
    if std::env::var_os("TERM_PROGRAM").is_none() {
        cmd.env("TERM_PROGRAM", "neural-junkie");
    }
    cmd.cwd(working_dir);

    let child = pair
        .slave
        .spawn_command(cmd)
        .map_err(|e| format!("Failed to spawn shell: {}", e))?;
    let process_id = child.process_id();

    let writer = pair
        .master
        .take_writer()
        .map_err(|e| format!("Failed to take PTY writer: {}", e))?;

    // Background reader thread: reads PTY output and emits events to the frontend
    let mut reader = pair
        .master
        .try_clone_reader()
        .map_err(|e| format!("Failed to clone PTY reader: {}", e))?;

    let session_id = id.clone();
    let handle = app_handle.clone();
    std::thread::spawn(move || {
        let mut buf = [0u8; 4096];
        loop {
            match reader.read(&mut buf) {
                Ok(0) => break,
                Ok(n) => {
                    let text = String::from_utf8_lossy(&buf[..n]).to_string();
                    let payload = serde_json::json!({
                        "id": session_id,
                        "data": text,
                    });
                    let _ = handle.emit("pty-output", payload);
                }
                Err(_) => break,
            }
        }
    });

    let sessions = app_handle.state::<PtySessions>();
    let mut guard = sessions.lock().unwrap();
    guard.insert(
        id,
        PtySession {
            writer,
            _child: child,
            pair,
            process_id,
        },
    );

    Ok(())
}

#[cfg(unix)]
fn pty_has_foreground_work(process_id: Option<u32>) -> bool {
    let Some(process_id) = process_id else {
        return false;
    };
    let output = Command::new("ps")
        .args(["-o", "pgid=", "-o", "tpgid=", "-p", &process_id.to_string()])
        .output();
    let Ok(output) = output else {
        return false;
    };
    let fields: Vec<_> = String::from_utf8_lossy(&output.stdout)
        .split_whitespace()
        .filter_map(|field| field.parse::<i64>().ok())
        .collect();
    matches!(fields.as_slice(), [shell_group, foreground_group] if *foreground_group > 0 && shell_group != foreground_group)
}

#[cfg(not(unix))]
fn pty_has_foreground_work(_process_id: Option<u32>) -> bool {
    // ConPTY does not expose a portable foreground process-group query.
    // Conservatively protect every connected Windows terminal session.
    true
}

#[tauri::command]
async fn get_pty_session_status(
    id: String,
    app_handle: tauri::AppHandle,
) -> Result<PtySessionStatus, String> {
    let sessions = app_handle.state::<PtySessions>();
    let guard = sessions.lock().unwrap();
    let session = guard
        .get(&id)
        .ok_or_else(|| format!("PTY session '{}' not found", id))?;
    Ok(PtySessionStatus {
        foreground_work: pty_has_foreground_work(session.process_id),
    })
}

#[tauri::command]
async fn write_pty_session(
    id: String,
    data: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    let sessions = app_handle.state::<PtySessions>();
    let mut guard = sessions.lock().unwrap();
    let session = guard
        .get_mut(&id)
        .ok_or_else(|| format!("PTY session '{}' not found", id))?;
    session
        .writer
        .write_all(data.as_bytes())
        .map_err(|e| format!("Failed to write to PTY: {}", e))?;
    session
        .writer
        .flush()
        .map_err(|e| format!("Failed to flush PTY: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn resize_pty_session(
    id: String,
    cols: u16,
    rows: u16,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    let sessions = app_handle.state::<PtySessions>();
    let guard = sessions.lock().unwrap();
    let session = guard
        .get(&id)
        .ok_or_else(|| format!("PTY session '{}' not found", id))?;
    session
        .pair
        .master
        .resize(PtySize {
            rows,
            cols,
            pixel_width: 0,
            pixel_height: 0,
        })
        .map_err(|e| format!("Failed to resize PTY: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn close_pty_session(id: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    let sessions = app_handle.state::<PtySessions>();
    let mut guard = sessions.lock().unwrap();
    guard.remove(&id);
    Ok(())
}

// ── One-off command execution (used for suggestion approve) ─────────────

#[tauri::command]
async fn execute_command(
    command: String,
    working_dir: Option<String>,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<CommandResult, String> {
    if !command_security::command_allowed(&command) {
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

#[tauri::command]
async fn open_browser_window(url: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    // Never render arbitrary http(s) pages inside Neural Junkie.
    // In-app HTML preview belongs to the web-browser pack workbench only.
    // Always hand off to the OS default browser.
    let trimmed = url.trim();
    if trimmed.is_empty() {
        return Err("empty URL".to_string());
    }
    app_handle
        .opener()
        .open_url(trimmed, None::<&str>)
        .map_err(|e| format!("Failed to open URL in system browser: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn close_browser_window(app_handle: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app_handle.get_webview_window("browser-popout") {
        window
            .close()
            .map_err(|e| format!("Failed to close browser window: {}", e))?;
    }
    Ok(())
}

#[tauri::command]
async fn capture_browser_screenshot(app_handle: tauri::AppHandle) -> Result<String, String> {
    // Tauri does not expose a built-in cross-platform screenshot API here.
    // We'll use JavaScript to capture the page content as an image

    if let Some(window) = app_handle.get_webview_window("embedded-browser") {
        // Use JavaScript to capture the page
        let js_code = r#"
            (async function() {
                try {
                    // Use html2canvas if available, otherwise return a data URL with page info
                    const canvas = document.createElement('canvas');
                    canvas.width = document.documentElement.scrollWidth;
                    canvas.height = document.documentElement.scrollHeight;
                    const ctx = canvas.getContext('2d');
                    ctx.fillStyle = 'white';
                    ctx.fillRect(0, 0, canvas.width, canvas.height);
                    ctx.fillStyle = 'black';
                    ctx.font = '16px Arial';
                    ctx.fillText('Screenshot: ' + window.location.href, 20, 50);
                    ctx.fillText('Captured at: ' + new Date().toISOString(), 20, 80);
                    return canvas.toDataURL('image/png');
                } catch (e) {
                    return 'error:' + e.message;
                }
            })();
        "#;

        match window.eval(js_code) {
            Ok(_) => {
                // For now, return a simple data URL indicating screenshot functionality
                // In production, you'd need to integrate a proper screenshot library
                Ok("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==".to_string())
            }
            Err(e) => Err(format!("Failed to execute screenshot script: {}", e)),
        }
    } else if let Some(_popout_window) = app_handle.get_webview_window("browser-popout") {
        // Same approach for pop-out window
        Ok("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==".to_string())
    } else {
        Err("No browser window available for screenshot".to_string())
    }
}

#[tauri::command]
async fn navigate_browser(url: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    // This would communicate with the browser window to navigate to the URL
    // For now, just emit an event that the frontend can listen to
    app_handle
        .emit("browser-navigate", &url)
        .map_err(|e| format!("Failed to emit navigation event: {}", e))?;
    Ok(())
}

#[tauri::command]
async fn create_embedded_browser(
    url: String,
    x: f64,
    y: f64,
    width: f64,
    height: f64,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Destroy existing embedded browser if it exists and wait for it to close
    // This prevents duplicate windows
    if let Some(window) = app_handle.get_webview_window("embedded-browser") {
        let _ = window.close();
        // Give a small delay to ensure the window is fully destroyed
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
    }

    // Double-check that window doesn't exist
    if app_handle.get_webview_window("embedded-browser").is_some() {
        return Err("Previous browser window still exists".to_string());
    }

    // Create new browser window positioned over the panel
    let _window = WebviewWindowBuilder::new(
        &app_handle,
        "embedded-browser",
        WebviewUrl::External(url.parse().map_err(|e| format!("Invalid URL: {}", e))?),
    )
    .title("Embedded Browser")
    .inner_size(width, height)
    .position(x, y)
    .resizable(false)
    .decorations(false)
    .always_on_top(false)
    .skip_taskbar(true)
    .focused(false) // Don't automatically focus the window
    .build()
    .map_err(|e| format!("Failed to create embedded browser: {}", e))?;

    // Explicitly ensure window doesn't have focus
    // On macOS, we need to handle focus more carefully
    #[cfg(target_os = "macos")]
    {
        // Try to keep main window in front - this is best effort on macOS
        // Get all windows and find the primary one (not embedded-browser)
        let windows = app_handle.webview_windows();
        for (label, _) in windows {
            if label != "embedded-browser" {
                if let Some(main_window) = app_handle.get_webview_window(label.as_str()) {
                    let _ = main_window.set_focus();
                    break;
                }
            }
        }
    }

    // Emit event that browser is ready
    app_handle
        .emit("browser-ready", &url)
        .map_err(|e| format!("Failed to emit browser ready event: {}", e))?;

    // The frontend owns page-load state for this secondary webview.
    // The frontend will handle page load state management

    Ok(())
}

#[tauri::command]
async fn update_browser_position(
    x: f64,
    y: f64,
    width: f64,
    height: f64,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if let Some(window) = app_handle.get_webview_window("embedded-browser") {
        window
            .set_position(tauri::LogicalPosition::new(x, y))
            .map_err(|e| format!("Failed to update browser position: {}", e))?;
        window
            .set_size(tauri::LogicalSize::new(width, height))
            .map_err(|e| format!("Failed to update browser size: {}", e))?;
    }
    Ok(())
}

#[tauri::command]
async fn navigate_embedded_browser(
    url: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if let Some(_window) = app_handle.get_webview_window("embedded-browser") {
        // Destroy the existing browser and create a new one with the new URL
        // This is more reliable than using eval() and provides proper navigation state
        let _ = destroy_embedded_browser(app_handle.clone()).await;

        // Emit navigation start event
        app_handle
            .emit("browser-navigation-start", &url)
            .map_err(|e| format!("Failed to emit navigation event: {}", e))?;

        // Note: The frontend will need to call create_embedded_browser with the new URL
        // This approach gives us better control over the navigation lifecycle
        Ok(())
    } else {
        Err("No embedded browser window found".to_string())
    }
}

#[tauri::command]
async fn destroy_embedded_browser(app_handle: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app_handle.get_webview_window("embedded-browser") {
        window
            .close()
            .map_err(|e| format!("Failed to close embedded browser: {}", e))?;
        // Wait a moment to ensure window is fully closed
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
    }
    Ok(())
}

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

const MAX_PACK_ZIP_BYTES: usize = 10 << 20; // 10 MiB — matches hub internal/packs

#[derive(Debug, Serialize, Deserialize)]
struct PackScaffoldRequest {
    output_dir: String,
    id: String,
    version: String,
    title: String,
    description: Option<String>,
    publisher: Option<String>,
    requires_packs: Vec<String>,
    capabilities: Vec<String>,
    settings_overlay: std::collections::HashMap<String, String>,
    workspace_guide: Option<String>,
    runbooks_glob: Option<String>,
    /// Optional sidebar chip label (max 3 characters).
    toolbar_chip_label: Option<String>,
    /// Optional pack-relative icon path (e.g. assets/icons/chip.png).
    toolbar_chip_icon: Option<String>,
}

/// Register an absolute path from a native file/folder picker (pack dev / custom install).
#[tauri::command]
fn register_pack_path(path: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    let trimmed = path.trim();
    if trimmed.is_empty() {
        return Err("empty path".into());
    }
    let allowlist = app_handle.state::<PackPathAllowlist>();
    allowlist_insert(allowlist.inner(), trimmed);
    Ok(())
}

/// Pick a pack directory via native folder dialog.
#[tauri::command]
fn pick_pack_directory(
    title: Option<String>,
    app_handle: tauri::AppHandle,
) -> Result<Option<String>, String> {
    let mut builder = app_handle.dialog().file();
    if let Some(t) = title {
        builder = builder.set_title(&t);
    }
    let picked = builder
        .blocking_pick_folder()
        .and_then(|p| p.into_path().ok())
        .map(|p| p.to_string_lossy().into_owned());
    if let Some(ref path) = picked {
        let allowlist = app_handle.state::<PackPathAllowlist>();
        allowlist_insert(allowlist.inner(), path);
    }
    Ok(picked)
}

/// Create scaffold pack.yaml and starter assets in output_dir.
#[tauri::command]
fn write_pack_scaffold(
    req: PackScaffoldRequest,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<PackPathAllowlist>();
    path_allowed(&allowed_roots, allowlist.inner(), &req.output_dir)?;
    let output_dir = req.output_dir.trim();
    if output_dir.is_empty() {
        return Err("output_dir required".into());
    }
    let id = req.id.trim();
    if id.is_empty() {
        return Err("id required".into());
    }
    let title = req.title.trim();
    if title.is_empty() {
        return Err("title required".into());
    }
    std::fs::create_dir_all(output_dir).map_err(|e| format!("create output dir: {}", e))?;
    let assets_dir = format!("{}/assets", output_dir);
    std::fs::create_dir_all(&assets_dir).map_err(|e| format!("create assets dir: {}", e))?;
    let runbooks_dir = format!("{}/assets/runbooks", output_dir);
    let _ = std::fs::create_dir_all(&runbooks_dir);

    let workspace_guide = req
        .workspace_guide
        .filter(|s| !s.trim().is_empty())
        .unwrap_or_else(|| "assets/WORKSPACE.md".to_string());
    let guide_path = format!("{}/{}", output_dir, workspace_guide);
    if let Some(parent) = std::path::Path::new(&guide_path).parent() {
        std::fs::create_dir_all(parent).map_err(|e| format!("create guide parent: {}", e))?;
    }
    if !std::path::Path::new(&guide_path).exists() {
        let guide_body = format!(
            "# {}\n\nWorkspace guide for the **{}** custom pack.\n\n## Layout\n\nDescribe expected folders and data layout here.\n",
            title, id
        );
        std::fs::write(&guide_path, guide_body)
            .map_err(|e| format!("write workspace guide: {}", e))?;
    }

    let mut caps: Vec<String> = req.capabilities;
    if !caps.iter().any(|c| c == "customer-pack") {
        caps.insert(0, "customer-pack".to_string());
    }
    let chip_label = req
        .toolbar_chip_label
        .as_ref()
        .map(|s| s.trim())
        .filter(|s| !s.is_empty());
    let chip_icon = req
        .toolbar_chip_icon
        .as_ref()
        .map(|s| s.trim())
        .filter(|s| !s.is_empty());
    let include_toolbar_chip = chip_label.is_some() || chip_icon.is_some();
    if include_toolbar_chip && !caps.iter().any(|c| c == "pack-toolbar") {
        caps.push("pack-toolbar".to_string());
    }
    let mut yaml = String::new();
    yaml.push_str(&format!("id: {}\n", id));
    yaml.push_str(&format!("version: \"{}\"\n", req.version.trim()));
    yaml.push_str(&format!("title: {}\n", title));
    if let Some(desc) = req.description.filter(|s| !s.trim().is_empty()) {
        yaml.push_str(&format!("description: >-\n  {}\n", desc.trim()));
    }
    if let Some(pub_name) = req.publisher.filter(|s| !s.trim().is_empty()) {
        yaml.push_str(&format!("publisher: {}\n", pub_name.trim()));
    }
    yaml.push_str("pack_kind: customer\n");
    yaml.push_str("layout_profile: team\n");
    yaml.push_str("capabilities:\n");
    for c in &caps {
        yaml.push_str(&format!("  - {}\n", c));
    }
    if include_toolbar_chip {
        yaml.push_str("capability_defs:\n");
        yaml.push_str("  pack-toolbar:\n");
        yaml.push_str("    kind: toolbar-chip\n");
        yaml.push_str("    ui:\n");
        yaml.push_str("      toolbar:\n");
        yaml.push_str(&format!("        id: {}-chip\n", id));
        if let Some(label) = chip_label {
            let short: String = label.chars().take(3).collect();
            yaml.push_str(&format!("        label: {}\n", short));
        }
        if let Some(icon) = chip_icon {
            yaml.push_str(&format!("        icon: {}\n", icon));
        }
    }
    if !req.requires_packs.is_empty() {
        yaml.push_str("requires_packs:\n");
        for p in &req.requires_packs {
            let p = p.trim();
            if !p.is_empty() {
                yaml.push_str(&format!("  - {}\n", p));
            }
        }
    }
    if !req.settings_overlay.is_empty() {
        yaml.push_str("settings_overlay:\n");
        for (k, v) in &req.settings_overlay {
            yaml.push_str(&format!("  {}: {}\n", k, v));
        }
    }
    yaml.push_str("assets:\n");
    yaml.push_str(&format!("  workspace_guide: {}\n", workspace_guide));
    if let Some(glob) = req.runbooks_glob.filter(|s| !s.trim().is_empty()) {
        yaml.push_str(&format!("  runbooks_glob: {}\n", glob.trim()));
    }

    let manifest_path = format!("{}/pack.yaml", output_dir);
    std::fs::write(&manifest_path, &yaml).map_err(|e| format!("write pack.yaml: {}", e))?;
    Ok(yaml)
}

/// Read pack.yaml from an absolute pack directory.
#[tauri::command]
fn read_pack_yaml_from_dir(
    absolute_dir: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<PackPathAllowlist>();
    path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
    let dir = absolute_dir.trim();
    if dir.is_empty() {
        return Err("empty directory".into());
    }
    let path = format!("{}/pack.yaml", dir);
    std::fs::read_to_string(&path).map_err(|e| format!("read {}: {}", path, e))
}

/// Write pack.yaml to an absolute pack directory.
#[tauri::command]
fn write_pack_yaml_to_dir(
    absolute_dir: String,
    yaml: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    let allowlist = app_handle.state::<PackPathAllowlist>();
    path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
    let dir = absolute_dir.trim();
    if dir.is_empty() {
        return Err("empty directory".into());
    }
    if yaml.trim().is_empty() {
        return Err("yaml content required".into());
    }
    let path = format!("{}/pack.yaml", dir);
    std::fs::write(&path, yaml).map_err(|e| format!("write {}: {}", path, e))
}

/// Zip a pack directory for release testing; returns base64 payload.
#[tauri::command]
fn zip_pack_directory(
    absolute_dir: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<PackPathAllowlist>();
    path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
    use std::fs::File;
    use std::io::Write;
    use zip::write::SimpleFileOptions;
    use zip::ZipWriter;

    let dir = absolute_dir.trim();
    if dir.is_empty() {
        return Err("empty directory".into());
    }
    let manifest = format!("{}/pack.yaml", dir);
    if !std::path::Path::new(&manifest).exists() {
        return Err(format!("pack.yaml not found in {}", dir));
    }
    let tmp = std::env::temp_dir().join(format!("nj-pack-{}.zip", uuid::Uuid::new_v4()));
    let file = File::create(&tmp).map_err(|e| format!("create zip: {}", e))?;
    let mut zip = ZipWriter::new(file);
    let options = SimpleFileOptions::default().compression_method(zip::CompressionMethod::Deflated);
    let dir_path = std::path::Path::new(dir);
    for entry in walkdir_for_pack(dir_path)? {
        let rel = entry
            .strip_prefix(dir_path)
            .map_err(|e| e.to_string())?
            .to_string_lossy()
            .replace('\\', "/");
        if entry.is_dir() {
            continue;
        }
        zip.start_file(rel, options)
            .map_err(|e| format!("zip start file: {}", e))?;
        let data = std::fs::read(&entry).map_err(|e| format!("read {}: {}", entry.display(), e))?;
        zip.write_all(&data)
            .map_err(|e| format!("zip write: {}", e))?;
    }
    zip.finish().map_err(|e| format!("zip finish: {}", e))?;
    let data = std::fs::read(&tmp).map_err(|e| format!("read zip: {}", e))?;
    let _ = std::fs::remove_file(&tmp);
    if data.len() > MAX_PACK_ZIP_BYTES {
        return Err(format!("pack zip exceeds {} bytes", MAX_PACK_ZIP_BYTES));
    }
    Ok(base64::Engine::encode(
        &base64::engine::general_purpose::STANDARD,
        data,
    ))
}

fn walkdir_for_pack(root: &std::path::Path) -> Result<Vec<std::path::PathBuf>, String> {
    let mut out = Vec::new();
    let mut stack = vec![root.to_path_buf()];
    while let Some(dir) = stack.pop() {
        let entries =
            std::fs::read_dir(&dir).map_err(|e| format!("read dir {}: {}", dir.display(), e))?;
        for entry in entries {
            let entry = entry.map_err(|e| e.to_string())?;
            let path = entry.path();
            if path.is_dir() {
                stack.push(path);
            } else {
                out.push(path);
            }
        }
    }
    Ok(out)
}

/// Read a custom pack zip as base64 (dialog paths are outside Tauri fs allowlist).
#[tauri::command]
fn read_pack_zip_base64(
    absolute_path: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<PackPathAllowlist>();
    path_allowed(&allowed_roots, allowlist.inner(), &absolute_path)?;
    let path = absolute_path.trim();
    if path.is_empty() {
        return Err("empty path".into());
    }
    let meta = std::fs::metadata(path).map_err(|e| format!("{}: {}", path, e))?;
    if !meta.is_file() {
        return Err(format!("not a file: {}", path));
    }
    if meta.len() as usize > MAX_PACK_ZIP_BYTES {
        return Err(format!("pack zip exceeds {} bytes", MAX_PACK_ZIP_BYTES));
    }
    let data = std::fs::read(path).map_err(|e| format!("read {}: {}", path, e))?;
    Ok(base64::Engine::encode(
        &base64::engine::general_purpose::STANDARD,
        data,
    ))
}

/// Read text files from absolute paths for chat prompt attachments (drag-and-drop from Finder).
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
type OllamaChild = Arc<Mutex<Option<std::process::Child>>>;

static SIDECAR_READY: AtomicBool = AtomicBool::new(false);
static SHUTTING_DOWN: AtomicBool = AtomicBool::new(false);

fn target_triple() -> &'static str {
    #[cfg(all(target_os = "macos", target_arch = "aarch64"))]
    return "aarch64-apple-darwin";
    #[cfg(all(target_os = "macos", target_arch = "x86_64"))]
    return "x86_64-apple-darwin";
    #[cfg(all(target_os = "linux", target_arch = "x86_64"))]
    return "x86_64-unknown-linux-gnu";
    #[cfg(all(target_os = "windows", target_arch = "x86_64"))]
    return "x86_64-pc-windows-msvc";
    #[allow(unreachable_code)]
    "unknown"
}

fn bundled_ollama_runtime_dir(app: &tauri::AppHandle) -> Option<PathBuf> {
    let triple = target_triple();
    // Prefer user-updated runtime (~/.neural-junkie/ollama-runtime/{triple}) over app resources.
    let user_runtime = default_home()
        .join(".neural-junkie")
        .join("ollama-runtime")
        .join(&triple);
    if bundled_ollama_binary(&user_runtime).exists() {
        return Some(user_runtime);
    }

    let rel = format!("ollama/{}", triple);
    if let Ok(p) = app.path().resolve(&rel, BaseDirectory::Resource) {
        if bundled_ollama_binary(&p).exists() {
            return Some(p);
        }
    }
    // Dev (`tauri dev`): fetch-ollama lays out under src-tauri/ollama/{triple}
    let dev = PathBuf::from(env!("CARGO_MANIFEST_DIR")).join(&rel);
    if bundled_ollama_binary(&dev).exists() {
        return Some(dev);
    }
    None
}

fn bundled_ollama_binary(runtime_dir: &std::path::Path) -> PathBuf {
    #[cfg(target_os = "windows")]
    {
        return runtime_dir.join("ollama.exe");
    }
    #[cfg(not(target_os = "windows"))]
    {
        let nested = runtime_dir.join("bin").join("ollama");
        if nested.exists() {
            return nested;
        }
        runtime_dir.join("ollama")
    }
}

fn system_ollama_on_path() -> bool {
    std::process::Command::new("ollama")
        .arg("--version")
        .stdout(Stdio::null())
        .stderr(Stdio::null())
        .status()
        .map(|s| s.success())
        .unwrap_or(false)
}

fn isolated_ollama_models_dir(app: &tauri::AppHandle) -> PathBuf {
    app.path()
        .app_data_dir()
        .ok()
        .map(|p| p.join("ollama-models"))
        .unwrap_or_else(|| default_home().join(".neural-junkie").join("ollama-models"))
}

/// Where bundled `ollama serve` stores models. Dev machines with a system Ollama install share ~/.ollama/models.
fn ollama_models_dir(app: &tauri::AppHandle) -> PathBuf {
    if let Ok(raw) = std::env::var("NJ_OLLAMA_MODELS") {
        let trimmed = raw.trim();
        if !trimmed.is_empty() {
            return PathBuf::from(trimmed);
        }
    }
    if system_ollama_on_path() {
        return default_home().join(".ollama").join("models");
    }
    isolated_ollama_models_dir(app)
}

fn ollama_health_client() -> reqwest::blocking::Client {
    reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(2))
        .build()
        .unwrap()
}

fn is_ollama_healthy() -> bool {
    let client = ollama_health_client();
    client
        .get("http://127.0.0.1:11434/api/tags")
        .send()
        .map(|r| r.status().is_success())
        .unwrap_or(false)
}

fn wait_for_ollama_health(timeout: std::time::Duration) -> bool {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if is_ollama_healthy() {
            return true;
        }
        std::thread::sleep(std::time::Duration::from_millis(300));
    }
    false
}

fn wait_for_ollama_stopped(timeout: std::time::Duration) -> bool {
    let start = Instant::now();
    while start.elapsed() < timeout {
        if !is_ollama_healthy() {
            return true;
        }
        std::thread::sleep(std::time::Duration::from_millis(200));
    }
    false
}

/// Stop any process listening on port (e.g. orphaned `ollama serve` / `nj-server` not tracked by Tauri).
fn kill_listeners_on_port(port: u16) {
    #[cfg(any(target_os = "macos", target_os = "linux"))]
    {
        let port_arg = format!(":{}", port);
        if let Ok(out) = Command::new("lsof").args(["-ti", &port_arg]).output() {
            for pid in String::from_utf8_lossy(&out.stdout).split_whitespace() {
                // Prefer graceful stop so Go hub can tear down pack sidecars; then force.
                let _ = Command::new("kill").arg(pid).status();
                std::thread::sleep(std::time::Duration::from_millis(150));
                let _ = Command::new("kill").args(["-9", pid]).status();
            }
        }
    }
}

fn stop_all_ollama_on_port() {
    kill_listeners_on_port(11434);
    let _ = wait_for_ollama_stopped(std::time::Duration::from_secs(8));
}

/// Default packaged hub listen port (matches desktop hubUrl DEFAULT_HUB_HTTP).
const DEFAULT_HUB_PORT: u16 = 18765;

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
    kill_listeners_on_port(port);
    let _ = wait_for_hub_stopped(std::time::Duration::from_secs(8));
}

fn bundled_ollama_version(binary: &std::path::Path) -> Option<String> {
    Command::new(binary)
        .arg("--version")
        .output()
        .ok()
        .map(|out| String::from_utf8_lossy(&out.stdout).trim().to_string())
        .filter(|s| !s.is_empty())
}

fn stop_bundled_ollama_child(ollama_state: &OllamaChild) {
    let mut guard = ollama_state.lock().unwrap();
    if let Some(mut child) = guard.take() {
        let _ = child.kill();
        let _ = child.wait();
    }
}

#[derive(Debug, Deserialize)]
struct SessionOllamaModelsFile {
    models: Vec<SessionOllamaModelRef>,
}

#[derive(Debug, Deserialize, Clone)]
struct SessionOllamaModelRef {
    endpoint: String,
    model: String,
}

fn session_ollama_models_path() -> PathBuf {
    default_home()
        .join(".neural-junkie")
        .join("session-ollama-models.json")
}

fn read_persisted_session_ollama_models() -> Vec<SessionOllamaModelRef> {
    let path = session_ollama_models_path();
    let raw = match std::fs::read_to_string(&path) {
        Ok(s) => s,
        Err(_) => return Vec::new(),
    };
    serde_json::from_str::<SessionOllamaModelsFile>(&raw)
        .map(|doc| {
            doc.models
                .into_iter()
                .filter(|m| !m.model.trim().is_empty())
                .collect()
        })
        .unwrap_or_default()
}

fn unload_ollama_model(endpoint: &str, model: &str) -> Result<(), String> {
    let endpoint = endpoint.trim().trim_end_matches('/');
    let endpoint = if endpoint.is_empty() {
        "http://127.0.0.1:11434"
    } else {
        endpoint
    };
    let model = model.trim();
    if model.is_empty() {
        return Err("empty model".into());
    }
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(15))
        .build()
        .map_err(|e| e.to_string())?;
    let body = serde_json::json!({
        "model": model,
        "keep_alive": 0,
    });
    let resp = client
        .post(format!("{endpoint}/api/generate"))
        .json(&body)
        .send()
        .map_err(|e| e.to_string())?;
    if !resp.status().is_success() {
        let status = resp.status();
        let text = resp.text().unwrap_or_default();
        return Err(format!("ollama unload {status}: {text}"));
    }
    Ok(())
}

/// Ask the hub to unload session models; fall back to the persisted session file + Ollama HTTP.
fn unload_session_ollama_models_on_exit() {
    let hub_base = std::env::var("NEURAL_JUNKIE_HUB_URL")
        .or_else(|_| std::env::var("VITE_NJ_HUB_URL"))
        .unwrap_or_else(|_| format!("http://127.0.0.1:{DEFAULT_HUB_PORT}"));
    let hub_base = hub_base.trim().trim_end_matches('/');
    let client = reqwest::blocking::Client::builder()
        .timeout(std::time::Duration::from_secs(35))
        .build();
    if let Ok(client) = client {
        if let Ok(resp) = client
            .post(format!("{hub_base}/api/ollama/unload-session"))
            .send()
        {
            if resp.status().is_success() {
                let _ = std::fs::remove_file(session_ollama_models_path());
                eprintln!("Unloaded Ollama session models via hub");
                return;
            }
        }
    }

    let models = read_persisted_session_ollama_models();
    if models.is_empty() {
        return;
    }
    let mut unloaded = 0usize;
    for entry in &models {
        match unload_ollama_model(&entry.endpoint, &entry.model) {
            Ok(()) => {
                unloaded += 1;
                eprintln!("Unloaded Ollama model {}", entry.model);
            }
            Err(err) => eprintln!(
                "Failed to unload Ollama model {}: {}",
                entry.model, err
            ),
        }
    }
    if unloaded > 0 {
        let _ = std::fs::remove_file(session_ollama_models_path());
    }
}

fn spawn_bundled_ollama(app: &tauri::AppHandle, ollama_state: &OllamaChild) -> Option<PathBuf> {
    let runtime_dir = bundled_ollama_runtime_dir(app)?;
    let binary = bundled_ollama_binary(&runtime_dir);
    if !binary.exists() {
        eprintln!(
            "Bundled Ollama runtime missing binary at {}",
            binary.display()
        );
        return None;
    }

    if is_ollama_healthy() {
        eprintln!("Ollama already running at http://127.0.0.1:11434");
        return Some(binary);
    }

    let models_dir = ollama_models_dir(app);
    if let Err(e) = std::fs::create_dir_all(&models_dir) {
        eprintln!("Failed to create Ollama models dir: {}", e);
        return None;
    }

    let mut cmd = Command::new(&binary);
    cmd.arg("serve");
    cmd.current_dir(&runtime_dir);
    cmd.env("OLLAMA_HOST", "127.0.0.1:11434");
    cmd.env("OLLAMA_MODELS", &models_dir);
    cmd.stdout(Stdio::null());
    cmd.stderr(Stdio::piped());

    match cmd.spawn() {
        Ok(mut child) => {
            if let Some(stderr) = child.stderr.take() {
                std::thread::spawn(move || {
                    use std::io::{BufRead, BufReader};
                    let reader = BufReader::new(stderr);
                    for line in reader.lines().flatten() {
                        eprintln!("[ollama] {}", line);
                    }
                });
            }
            *ollama_state.lock().unwrap() = Some(child);
            if wait_for_ollama_health(std::time::Duration::from_secs(45)) {
                eprintln!("Bundled Ollama server ready at http://127.0.0.1:11434");
                Some(binary)
            } else {
                eprintln!("Bundled Ollama started but health check timed out");
                Some(binary)
            }
        }
        Err(e) => {
            eprintln!("Failed to start bundled Ollama: {}", e);
            None
        }
    }
}

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
    if let Some(models_dir) = ollama_models_dir(app).to_str() {
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

#[derive(Debug, Serialize)]
struct OllamaRuntimeStatus {
    installed: bool,
    bundled: bool,
    running: bool,
    managed: bool,
    version: Option<String>,
}

#[tauri::command]
fn get_ollama_runtime_status(
    app: tauri::AppHandle,
    ollama_state: tauri::State<OllamaChild>,
) -> Result<OllamaRuntimeStatus, String> {
    let runtime_dir = bundled_ollama_runtime_dir(&app);
    let bundled = runtime_dir
        .as_ref()
        .map(|dir| bundled_ollama_binary(dir).exists())
        .unwrap_or(false);
    let version = runtime_dir.as_ref().and_then(|dir| {
        let binary = bundled_ollama_binary(dir);
        if binary.exists() {
            bundled_ollama_version(&binary)
        } else {
            None
        }
    });
    let managed = {
        let mut guard = ollama_state.lock().unwrap();
        if let Some(child) = guard.as_mut() {
            child.try_wait().ok().flatten().is_none()
        } else {
            false
        }
    };
    let running = is_ollama_healthy();
    Ok(OllamaRuntimeStatus {
        installed: bundled || running,
        bundled,
        running,
        managed,
        version,
    })
}

#[tauri::command]
fn start_bundled_ollama(
    app: tauri::AppHandle,
    ollama_state: tauri::State<OllamaChild>,
) -> Result<(), String> {
    if is_ollama_healthy() {
        return Ok(());
    }
    let state = ollama_state.inner().clone();
    spawn_bundled_ollama(&app, &state)
        .ok_or_else(|| "Bundled Ollama runtime is not available in this build".to_string())?;
    if !is_ollama_healthy() {
        return Err("Ollama started but health check failed".into());
    }
    Ok(())
}

#[tauri::command]
fn stop_bundled_ollama(
    app: tauri::AppHandle,
    ollama_state: tauri::State<OllamaChild>,
) -> Result<(), String> {
    let bundled = bundled_ollama_runtime_dir(&app)
        .map(|dir| bundled_ollama_binary(&dir).exists())
        .unwrap_or(false);
    if !bundled {
        return Err("Bundled Ollama runtime is not available in this build".into());
    }
    let had_child = ollama_state.lock().unwrap().is_some();
    stop_bundled_ollama_child(ollama_state.inner());
    if !had_child && is_ollama_healthy() {
        return Err("Ollama is not managed by the desktop shell in this session".into());
    }
    Ok(())
}

#[tauri::command]
fn restart_bundled_ollama(
    app: tauri::AppHandle,
    ollama_state: tauri::State<OllamaChild>,
) -> Result<(), String> {
    stop_bundled_ollama_child(ollama_state.inner());
    std::thread::sleep(std::time::Duration::from_millis(400));
    if is_ollama_healthy() {
        stop_all_ollama_on_port();
    }
    if is_ollama_healthy() {
        return Err(
            "Ollama is still running on port 11434 (stop `ollama serve` or quit other NJ instances, then retry)"
                .into(),
        );
    }
    let state = ollama_state.inner().clone();
    spawn_bundled_ollama(&app, &state)
        .ok_or_else(|| "Bundled Ollama runtime is not available in this build".to_string())?;
    if !wait_for_ollama_health(std::time::Duration::from_secs(45)) {
        return Err("Ollama restarted but health check timed out".into());
    }
    Ok(())
}

fn shutdown_managed_processes(
    pty_sessions: &PtySessions,
    sidecar_state: &SidecarChild,
    ollama_state: &OllamaChild,
) {
    if !begin_shutdown(&SHUTTING_DOWN) {
        return;
    }

    // Soft-unload models NJ loaded before killing the hub (which may be SIGKILL'd).
    unload_session_ollama_models_on_exit();

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
    stop_bundled_ollama_child(ollama_state);
    SIDECAR_READY.store(false, Ordering::SeqCst);
}

fn begin_shutdown(flag: &AtomicBool) -> bool {
    !flag.swap(true, Ordering::SeqCst)
}

#[tauri::command]
fn prepare_for_update(app: tauri::AppHandle) -> Result<(), String> {
    let pty_sessions = app.state::<PtySessions>().inner().clone();
    let sidecar_state = app.state::<SidecarChild>().inner().clone();
    let ollama_state = app.state::<OllamaChild>().inner().clone();
    shutdown_managed_processes(&pty_sessions, &sidecar_state, &ollama_state);
    Ok(())
}

/// Read ~/.neural-junkie/bootstrap.token for local admin unlock (loopback dev only).
#[tauri::command]
fn read_hub_bootstrap_token() -> Result<String, String> {
    let path = dirs::home_dir()
        .ok_or_else(|| "home directory not found".to_string())?
        .join(".neural-junkie")
        .join("bootstrap.token");
    std::fs::read_to_string(&path)
        .map(|s| s.trim().to_string())
        .map_err(|e| format!("could not read {}: {}", path.display(), e))
}

fn machine_credential_key() -> [u8; 32] {
    use sha2::{Digest, Sha256};
    let home = dirs::home_dir()
        .map(|p| p.to_string_lossy().to_string())
        .unwrap_or_else(|| "/".to_string());
    let user = std::env::var("USER")
        .or_else(|_| std::env::var("USERNAME"))
        .unwrap_or_else(|_| "user".to_string());
    let mut hasher = Sha256::new();
    hasher.update(b"neural-junkie-credential-v1:");
    hasher.update(home.as_bytes());
    hasher.update(user.as_bytes());
    hasher.finalize().into()
}

/// Encrypt a JSON credential blob for storage in Tauri plugin-store (machine-bound).
#[tauri::command]
fn encrypt_credential_blob(plaintext: String) -> Result<String, String> {
    use aes_gcm::aead::{Aead, KeyInit};
    use aes_gcm::{Aes256Gcm, Nonce};
    use rand::RngCore;
    let key = machine_credential_key();
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| e.to_string())?;
    let mut nonce_bytes = [0u8; 12];
    rand::thread_rng().fill_bytes(&mut nonce_bytes);
    let nonce = Nonce::from_slice(&nonce_bytes);
    let ciphertext = cipher
        .encrypt(nonce, plaintext.as_bytes())
        .map_err(|e| e.to_string())?;
    let mut out = nonce_bytes.to_vec();
    out.extend(ciphertext);
    Ok(base64::Engine::encode(
        &base64::engine::general_purpose::STANDARD,
        out,
    ))
}

/// Decrypt a blob produced by encrypt_credential_blob.
#[tauri::command]
fn decrypt_credential_blob(blob: String) -> Result<String, String> {
    use aes_gcm::aead::{Aead, KeyInit};
    use aes_gcm::{Aes256Gcm, Nonce};
    let raw = base64::Engine::decode(&base64::engine::general_purpose::STANDARD, blob.trim())
        .map_err(|e| e.to_string())?;
    if raw.len() < 13 {
        return Err("ciphertext too short".into());
    }
    let key = machine_credential_key();
    let cipher = Aes256Gcm::new_from_slice(&key).map_err(|e| e.to_string())?;
    let nonce = Nonce::from_slice(&raw[..12]);
    let plain = cipher
        .decrypt(nonce, &raw[12..])
        .map_err(|e| format!("decrypt failed: {}", e))?;
    String::from_utf8(plain).map_err(|e| e.to_string())
}

fn main() {
    let pty_sessions: PtySessions = Arc::new(Mutex::new(HashMap::new()));
    let sidecar_state: SidecarChild = Arc::new(Mutex::new(None));
    let ollama_state: OllamaChild = Arc::new(Mutex::new(None));
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
            let ollama_state = app.state::<OllamaChild>().inner().clone();
            let app_handle = app.handle().clone();

            std::thread::spawn(move || {
                // Only spawn sidecar in production builds; in dev the server
                // is started separately via `make server` or `make refresh`.
                if cfg!(debug_assertions) {
                    // Dev: start bundled Ollama when fetch-ollama artifacts exist (parity with production).
                    if !is_ollama_healthy() {
                        if let Some(bin) = spawn_bundled_ollama(&app_handle, &ollama_state) {
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

                let bundled_ollama = if is_ollama_healthy() {
                    bundled_ollama_runtime_dir(&app_handle)
                        .map(|dir| bundled_ollama_binary(&dir))
                        .filter(|bin| bin.exists())
                } else {
                    spawn_bundled_ollama(&app_handle, &ollama_state)
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
            register_pack_path,
            pick_pack_directory,
            write_pack_scaffold,
            read_pack_yaml_from_dir,
            write_pack_yaml_to_dir,
            zip_pack_directory,
            read_pack_zip_base64,
            read_prompt_attachment_paths,
            execute_command,
            create_pty_session,
            get_pty_session_status,
            write_pty_session,
            resize_pty_session,
            close_pty_session,
            decode_scan_well_tiff,
            open_markdown_preview,
            open_browser_window,
            close_browser_window,
            capture_browser_screenshot,
            navigate_browser,
            create_embedded_browser,
            update_browser_position,
            navigate_embedded_browser,
            destroy_embedded_browser,
            get_server_status,
            get_ollama_runtime_status,
            start_bundled_ollama,
            stop_bundled_ollama,
            restart_bundled_ollama,
            prepare_for_update,
            encrypt_credential_blob,
            decrypt_credential_blob,
            read_hub_bootstrap_token
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
