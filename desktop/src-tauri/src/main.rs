// Prevents additional console window on Windows in release, DO NOT REMOVE!!
#![cfg_attr(not(debug_assertions), windows_subsystem = "windows")]

use serde::{Deserialize, Serialize};
use std::process::{Command, Stdio};
use std::time::Instant;
use std::sync::{Arc, Mutex};
use std::collections::HashMap;
use tauri::Manager;
use std::path::PathBuf;
use std::hash::{Hash, Hasher};

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


#[derive(Debug, Clone)]
struct ShellSession {
    id: String,
    working_dir: PathBuf,
}

impl ShellSession {
    fn new() -> Self {
        Self {
            id: uuid::Uuid::new_v4().to_string(),
            working_dir: std::env::current_dir().unwrap_or_else(|_| PathBuf::from(".")),
        }
    }
}

#[tauri::command]
async fn execute_command(
    command: String,
    working_dir: Option<String>,
    app_handle: tauri::AppHandle,
) -> Result<CommandResult, String> {
    let start_time = Instant::now();
    let command_id = uuid::Uuid::new_v4().to_string();
    
    // Split command into parts for execution
    let parts: Vec<&str> = if cfg!(target_os = "windows") {
        // On Windows, use cmd /c
        vec!["cmd", "/c", &command]
    } else {
        // On Unix-like systems, use sh -c
        vec!["sh", "-c", &command]
    };

    let mut cmd = Command::new(parts[0]);
    cmd.args(&parts[1..]);
    cmd.stdout(Stdio::piped());
    cmd.stderr(Stdio::piped());

    // Set working directory if provided
    if let Some(dir) = working_dir {
        cmd.current_dir(dir);
    }

    // Execute the command
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

    // Emit the result as an event
    app_handle
        .emit_all("command-executed", &result)
        .map_err(|e| format!("Failed to emit event: {}", e))?;

    Ok(result)
}

// Global shell session storage
type ShellSessions = Arc<Mutex<HashMap<String, ShellSession>>>;

#[tauri::command]
async fn start_shell_session(
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let session = ShellSession::new();
    let session_id = session.id.clone();
    
    // Store the session in the global state
    let sessions = app_handle.state::<ShellSessions>();
    let mut sessions_guard = sessions.lock().unwrap();
    sessions_guard.insert(session_id.clone(), session);
    
    Ok(session_id)
}

#[tauri::command]
async fn execute_in_session(
    command: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Get the default session (for now, we'll use a single global session)
    let sessions = app_handle.state::<ShellSessions>();
    let mut sessions_guard = sessions.lock().unwrap();
    
    // Use a default session ID or create one if none exists
    let default_session_id = "default".to_string();
    let session = sessions_guard.entry(default_session_id.clone()).or_insert_with(|| ShellSession::new());
    
    // Parse the command to handle cd commands specially
    let trimmed_command = command.trim();
    let (command_to_execute, _new_working_dir) = if trimmed_command.starts_with("cd ") {
        let path = trimmed_command[3..].trim();
        let new_path = if path.is_empty() || path == "~" {
            PathBuf::from(std::env::var("HOME").unwrap_or_else(|_| "/".to_string()))
        } else if path.starts_with("~/") {
            let home = std::env::var("HOME").unwrap_or_else(|_| "/".to_string());
            PathBuf::from(home).join(&path[2..])
        } else if path.starts_with('/') {
            PathBuf::from(path)
        } else {
            session.working_dir.join(path)
        };

        let resolved = std::fs::canonicalize(&new_path)
            .map_err(|e| format!("cd: {}: {}", new_path.display(), e))?;

        session.working_dir = resolved.clone();
        
        (None, Some(resolved))
    } else {
        // For other commands, execute them in the current working directory
        (Some(trimmed_command.to_string()), None)
    };
    
    // Execute the command if it's not a cd command
    if let Some(cmd) = command_to_execute {
        let parts: Vec<&str> = if cfg!(target_os = "windows") {
            vec!["cmd", "/c", &cmd]
        } else {
            vec!["sh", "-c", &cmd]
        };

        let mut process = Command::new(parts[0]);
        process.args(&parts[1..]);
        process.current_dir(&session.working_dir);
        process.stdout(Stdio::piped());
        process.stderr(Stdio::piped());

        let output = process.output().map_err(|e| format!("Failed to execute command: {}", e))?;
        
        let stdout = String::from_utf8_lossy(&output.stdout).to_string();
        let stderr = String::from_utf8_lossy(&output.stderr).to_string();
        
        // Emit output events
        if !stdout.is_empty() {
            let _ = app_handle.emit_all("shell-output", &stdout);
        }
        if !stderr.is_empty() {
            let _ = app_handle.emit_all("shell-error", &stderr);
        }
    } else {
        // For cd commands, emit a message showing the new directory
        let _ = app_handle.emit_all("shell-output", &format!("Changed directory to: {}\n", session.working_dir.display()));
    }

    Ok(())
}

#[tauri::command]
async fn get_session_cwd(
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    // Get the current working directory from the session
    let sessions = app_handle.state::<ShellSessions>();
    let sessions_guard = sessions.lock().unwrap();
    
    // Use the default session
    let default_session_id = "default".to_string();
    if let Some(session) = sessions_guard.get(&default_session_id) {
        Ok(session.working_dir.to_string_lossy().to_string())
    } else {
        // Fallback to current directory if no session exists
        let cwd = std::env::current_dir()
            .map_err(|e| format!("Failed to get current directory: {}", e))?;
        Ok(cwd.to_string_lossy().to_string())
    }
}

#[tauri::command]
async fn open_browser_window(
    url: String,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    // Create unique window ID for browser
    let window_id = "browser-popout";
    
    // Check if window already exists
    if app_handle.get_window(&window_id).is_some() {
        // Focus existing window
        if let Some(window) = app_handle.get_window(&window_id) {
            let _ = window.set_focus();
        }
        return Ok(());
    }
    
    // Create new browser window with webview
    let window = tauri::WindowBuilder::new(
        &app_handle,
        &*window_id,
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
    let window = tauri::WindowBuilder::new(
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
                if let Some(main_window) = app_handle.get_window(&label) {
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
    if app_handle.get_window(&window_id).is_some() {
        // Focus existing window
        if let Some(window) = app_handle.get_window(&window_id) {
            let _ = window.set_focus();
        }
        return Ok(());
    }
    
    // Create new window
    let window = tauri::WindowBuilder::new(
        &app_handle,
        &window_id,
        tauri::WindowUrl::App(format!(
            "?preview=true&workspace={}&path={}",
            urlencoding::encode(&workspace_id),
            urlencoding::encode(&file_path)
        ).into())
    )
    .title(&format!("{} - Markdown Preview", filename))
    .inner_size(800.0, 600.0)
    .min_inner_size(400.0, 300.0)
    .resizable(true)
    .center()
    .build()
    .map_err(|e| format!("Failed to create window: {}", e))?;
    
    let _ = window.set_focus();
    Ok(())
}

fn main() {
    tauri::Builder::default()
        .plugin(tauri_plugin_store::Builder::default().build())
        .manage(Arc::new(Mutex::new(HashMap::<String, ShellSession>::new())))
        .invoke_handler(tauri::generate_handler![
            execute_command,
            start_shell_session,
            execute_in_session,
            get_session_cwd,
            open_markdown_preview,
            open_browser_window,
            close_browser_window,
            capture_browser_screenshot,
            navigate_browser,
            create_embedded_browser,
            update_browser_position,
            navigate_embedded_browser,
            destroy_embedded_browser
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

