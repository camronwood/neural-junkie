// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use std::process::{Command, Stdio};
use std::time::Instant;
use std::sync::{Arc, Mutex};
use std::collections::HashMap;
use std::io::{Read, Write};
use tauri::Manager;
use std::path::PathBuf;
use std::hash::{Hash, Hasher};
use portable_pty::{CommandBuilder, PtySize, native_pty_system};

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
}

type PtySessions = Arc<Mutex<HashMap<String, PtySession>>>;

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
    // Launch as interactive login shell
    cmd.arg("-l");
    cmd.cwd(working_dir);

    let child = pair
        .slave
        .spawn_command(cmd)
        .map_err(|e| format!("Failed to spawn shell: {}", e))?;

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
                    let _ = handle.emit_all("pty-output", payload);
                }
                Err(_) => break,
            }
        }
    });

    let sessions = app_handle.state::<PtySessions>();
    let mut guard = sessions.lock().unwrap();
    guard.insert(id, PtySession { writer, _child: child, pair });

    Ok(())
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
async fn close_pty_session(
    id: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
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
    app_handle: tauri::AppHandle,
) -> Result<CommandResult, String> {
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

    let output = cmd.output().map_err(|e| format!("Failed to execute command: {}", e))?;

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
        .emit_all("command-executed", &result)
        .map_err(|e| format!("Failed to emit event: {}", e))?;

    Ok(result)
}

#[tauri::command]
async fn open_browser_window(
    url: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Create unique window ID for browser
    let window_id = "browser-popout";
    
    // Check if window already exists
    if app_handle.get_window(window_id).is_some() {
        // Focus existing window
        if let Some(window) = app_handle.get_window(window_id) {
            let _ = window.set_focus();
        }
        return Ok(());
    }
    
    // Create new browser window with webview
    let window = tauri::WindowBuilder::new(
        &app_handle,
        window_id,
        tauri::WindowUrl::External(url.parse().map_err(|e| format!("Invalid URL: {}", e))?)
    )
    .title("Neural Junkie - Browser")
    .inner_size(1200.0, 800.0)
    .min_inner_size(800.0, 600.0)
    .resizable(true)
    .center()
    .build()
    .map_err(|e| format!("Failed to create browser window: {}", e))?;
    
    let _ = window.set_focus();
    Ok(())
}

#[tauri::command]
async fn close_browser_window(
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if let Some(window) = app_handle.get_window("browser-popout") {
        window.close().map_err(|e| format!("Failed to close browser window: {}", e))?;
    }
    Ok(())
}

#[tauri::command]
async fn capture_browser_screenshot(
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    // Note: Tauri 1.x doesn't have built-in screenshot API
    // We'll use JavaScript to capture the page content as an image
    
    if let Some(window) = app_handle.get_window("embedded-browser") {
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
            Err(e) => Err(format!("Failed to execute screenshot script: {}", e))
        }
    } else if let Some(_popout_window) = app_handle.get_window("browser-popout") {
        // Same approach for pop-out window
        Ok("data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==".to_string())
    } else {
        Err("No browser window available for screenshot".to_string())
    }
}

#[tauri::command]
async fn navigate_browser(
    url: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // This would communicate with the browser window to navigate to the URL
    // For now, just emit an event that the frontend can listen to
    app_handle
        .emit_all("browser-navigate", &url)
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
    if let Some(window) = app_handle.get_window("embedded-browser") {
        let _ = window.close();
        // Give a small delay to ensure the window is fully destroyed
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
    }
    
    // Double-check that window doesn't exist
    if app_handle.get_window("embedded-browser").is_some() {
        return Err("Previous browser window still exists".to_string());
    }
    
    // Create new browser window positioned over the panel
    let _window = tauri::WindowBuilder::new(
        &app_handle,
        "embedded-browser",
        tauri::WindowUrl::External(url.parse().map_err(|e| format!("Invalid URL: {}", e))?)
    )
    .title("Embedded Browser")
    .inner_size(width, height)
    .position(x, y)
    .resizable(false)
    .decorations(false)
    .always_on_top(false)
    .skip_taskbar(true)
    .focused(false)  // Don't automatically focus the window
    .build()
    .map_err(|e| format!("Failed to create embedded browser: {}", e))?;
    
    // Explicitly ensure window doesn't have focus
    // On macOS, we need to handle focus more carefully
    #[cfg(target_os = "macos")]
    {
        // Try to keep main window in front - this is best effort on macOS
        // Get all windows and find the primary one (not embedded-browser)
        let windows = app_handle.windows();
        for (label, _) in windows {
            if label != "embedded-browser" {
                if let Some(main_window) = app_handle.get_window(label.as_str()) {
                    let _ = main_window.set_focus();
                    break;
                }
            }
        }
    }
    
    // Emit event that browser is ready
    app_handle.emit_all("browser-ready", &url)
        .map_err(|e| format!("Failed to emit browser ready event: {}", e))?;
    
    // Note: Tauri 1.x doesn't have on_page_load events
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
    if let Some(window) = app_handle.get_window("embedded-browser") {
        window.set_position(tauri::LogicalPosition::new(x, y))
            .map_err(|e| format!("Failed to update browser position: {}", e))?;
        window.set_size(tauri::LogicalSize::new(width, height))
            .map_err(|e| format!("Failed to update browser size: {}", e))?;
    }
    Ok(())
}

#[tauri::command]
async fn navigate_embedded_browser(
    url: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if let Some(_window) = app_handle.get_window("embedded-browser") {
        // Destroy the existing browser and create a new one with the new URL
        // This is more reliable than using eval() and provides proper navigation state
        let _ = destroy_embedded_browser(app_handle.clone()).await;
        
        // Emit navigation start event
        app_handle.emit_all("browser-navigation-start", &url)
            .map_err(|e| format!("Failed to emit navigation event: {}", e))?;
        
        // Note: The frontend will need to call create_embedded_browser with the new URL
        // This approach gives us better control over the navigation lifecycle
        Ok(())
    } else {
        Err("No embedded browser window found".to_string())
    }
}

#[tauri::command]
async fn destroy_embedded_browser(
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if let Some(window) = app_handle.get_window("embedded-browser") {
        window.close().map_err(|e| format!("Failed to close embedded browser: {}", e))?;
        // Wait a moment to ensure window is fully closed
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
    }
    Ok(())
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
    if app_handle.get_window(window_id.as_str()).is_some() {
        // Focus existing window
        if let Some(window) = app_handle.get_window(window_id.as_str()) {
            let _ = window.set_focus();
        }
        return Ok(());
    }
    
    // Create new window
    let window = tauri::WindowBuilder::new(
        &app_handle,
        window_id.as_str(),
        tauri::WindowUrl::App(format!(
            "?preview=true&workspace={}&path={}",
            urlencoding::encode(&workspace_id),
            urlencoding::encode(&file_path)
        ).into())
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

type SidecarChild = Arc<Mutex<Option<tauri::api::process::CommandChild>>>;

static SIDECAR_READY: AtomicBool = AtomicBool::new(false);

fn dev_hub_health_url() -> String {
    let base = std::env::var("NEURAL_JUNKIE_HUB_URL")
        .or_else(|_| std::env::var("VITE_NJ_HUB_URL"))
        .unwrap_or_else(|_| "http://localhost:18765".to_string());
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

fn spawn_sidecar() -> Result<tauri::api::process::CommandChild, String> {
    let (mut rx, child) = tauri::api::process::Command::new_sidecar("nj-server")
        .map_err(|e| format!("Failed to create sidecar command: {}", e))?
        .spawn()
        .map_err(|e| format!("Failed to spawn sidecar: {}", e))?;

    // Drain sidecar stdout/stderr in background so pipe buffers don't fill
    std::thread::spawn(move || {
        use tauri::api::process::CommandEvent;
        while let Some(event) = rx.blocking_recv() {
            match event {
                CommandEvent::Stdout(line) => eprintln!("[nj-server] {}", line),
                CommandEvent::Stderr(line) => eprintln!("[nj-server err] {}", line),
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

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::default().build())
        .manage(Arc::new(Mutex::new(HashMap::<String, PtySession>::new())) as PtySessions)
        .manage(Arc::new(Mutex::new(None::<tauri::api::process::CommandChild>)) as SidecarChild)
        .setup(|app| {
            let sidecar_state = app.state::<SidecarChild>().inner().clone();
            let app_handle = app.handle();

            std::thread::spawn(move || {
                // Only spawn sidecar in production builds; in dev the server
                // is started separately via `make server` or `make refresh`.
                if cfg!(debug_assertions) {
                    // In dev mode, just poll for an already-running server (longer window: first Rust build + hub start).
                    if wait_for_server_health(std::time::Duration::from_secs(120)) {
                        SIDECAR_READY.store(true, Ordering::Relaxed);
                        let _ = app_handle.emit_all("server-ready", true);
                    } else {
                        let msg = format!(
                            "Neural Junkie hub not healthy at {}. From neural-junkie run: make server. If port 18765 is in use, set NEURAL_JUNKIE_HUB_URL and VITE_NJ_HUB_URL to match (e.g. http://127.0.0.1:18766) and start the hub with -addr :18766.",
                            dev_hub_health_url()
                        );
                        let _ = app_handle.emit_all("server-error", msg);
                    }
                    return;
                }

                match spawn_sidecar() {
                    Ok(child) => {
                        *sidecar_state.lock().unwrap() = Some(child);

                        if wait_for_server_health(std::time::Duration::from_secs(30)) {
                            SIDECAR_READY.store(true, Ordering::Relaxed);
                            let _ = app_handle.emit_all("server-ready", true);
                        } else {
                            let _ = app_handle.emit_all("server-error", "Server started but health check timed out");
                        }
                    }
                    Err(e) => {
                        eprintln!("Failed to start sidecar: {}", e);
                        let _ = app_handle.emit_all("server-error", e);
                    }
                }
            });

            Ok(())
        })
        .on_window_event(|event| {
            if let tauri::WindowEvent::Destroyed = event.event() {
                let sidecar = event.window().state::<SidecarChild>();
                let child = sidecar.lock().unwrap().take();
                if let Some(child) = child {
                    let _ = child.kill();
                }
            }
        })
        .invoke_handler(tauri::generate_handler![
            execute_command,
            create_pty_session,
            write_pty_session,
            resize_pty_session,
            close_pty_session,
            open_markdown_preview,
            open_browser_window,
            close_browser_window,
            capture_browser_screenshot,
            navigate_browser,
            create_embedded_browser,
            update_browser_position,
            navigate_embedded_browser,
            destroy_embedded_browser,
            get_server_status
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

