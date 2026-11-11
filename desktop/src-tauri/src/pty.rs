//! Embedded terminal PTY session management (extraction loop 2 from main.rs).

use portable_pty::{native_pty_system, CommandBuilder, PtySize};
use serde::Serialize;
use std::collections::HashMap;
use std::io::{Read, Write};
use std::path::PathBuf;
use std::process::Command;
use std::sync::{Arc, Mutex};
use tauri::{Emitter, Manager};

use crate::command_security;
use crate::path_security;

// ── PTY session management ──────────────────────────────────────────────

struct PtySession {
    writer: Box<dyn Write + Send>,
    _child: Box<dyn portable_pty::Child + Send>,
    pair: portable_pty::PtyPair,
    process_id: Option<u32>,
}

pub type PtySessions = Arc<Mutex<HashMap<String, PtySession>>>;

#[derive(Debug, Serialize)]
struct PtySessionStatus {
    foreground_work: bool,
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
pub async fn create_pty_session(
    id: String,
    cwd: Option<String>,
    cols: Option<u16>,
    rows: Option<u16>,
    allowed_roots: Option<Vec<String>>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if crate::app_is_shutting_down() {
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

    let roots = allowed_roots.unwrap_or_default();
    let working_dir = match cwd.as_deref() {
        None | Some("~") | Some("") => {
            if roots.is_empty() {
                default_home()
            } else {
                path_security::within_any_root(&roots, &default_home().to_string_lossy())
                    .unwrap_or_else(|_| PathBuf::from(roots[0].trim()))
            }
        }
        Some(p) if p.starts_with("~/") => {
            let expanded = default_home().join(&p[2..]);
            if roots.is_empty() {
                expanded
            } else {
                path_security::within_any_root(&roots, &expanded.to_string_lossy())?
            }
        }
        Some(p) => {
            if roots.is_empty() {
                // Require explicit roots for non-home absolute/relative cwd.
                return Err("no workspace roots configured for PTY cwd".into());
            }
            path_security::within_any_root(&roots, p)?
        }
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
pub async fn get_pty_session_status(
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

/// Write to a PTY. When `agent_driven` is true, apply hard command denylist.
#[tauri::command]
pub async fn write_pty_session(
    id: String,
    data: String,
    agent_driven: Option<bool>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    if agent_driven.unwrap_or(false) {
        let line = data.trim();
        // Strip trailing newline for policy check; reject hard-denied agent commands.
        let cmd = line.trim_end_matches('\n').trim();
        if !cmd.is_empty() && command_security::command_hard_denied(cmd) {
            return Err(format!(
                "command not allowed: {}",
                cmd.lines().next().unwrap_or(cmd)
            ));
        }
    }
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
pub async fn resize_pty_session(
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
pub async fn close_pty_session(id: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    let sessions = app_handle.state::<PtySessions>();
    let mut guard = sessions.lock().unwrap();
    guard.remove(&id);
    Ok(())
}

