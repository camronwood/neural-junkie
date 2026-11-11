//! Bundled Ollama runtime management (extraction from main.rs).

use serde::{Deserialize, Serialize};
use std::path::PathBuf;
use std::process::{Command, Stdio};
use std::sync::{Arc, Mutex};
use std::time::Instant;
use tauri::path::BaseDirectory;
use tauri::Manager;

fn default_home() -> PathBuf {
    dirs::home_dir().unwrap_or_else(|| PathBuf::from("/"))
}

pub(crate) type OllamaChild = Arc<Mutex<Option<std::process::Child>>>;

pub(crate) fn target_triple() -> &'static str {
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

pub(crate) fn bundled_ollama_runtime_dir(app: &tauri::AppHandle) -> Option<PathBuf> {
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

pub(crate) fn bundled_ollama_binary(runtime_dir: &std::path::Path) -> PathBuf {
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
pub(crate) fn ollama_models_dir(app: &tauri::AppHandle) -> PathBuf {
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

pub(crate) fn is_ollama_healthy() -> bool {
    let client = ollama_health_client();
    client
        .get("http://127.0.0.1:11434/api/tags")
        .send()
        .map(|r| r.status().is_success())
        .unwrap_or(false)
}

pub(crate) fn wait_for_ollama_health(timeout: std::time::Duration) -> bool {
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
pub(crate) fn kill_listeners_on_port(port: u16) {
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

fn bundled_ollama_version(binary: &std::path::Path) -> Option<String> {
    Command::new(binary)
        .arg("--version")
        .output()
        .ok()
        .map(|out| String::from_utf8_lossy(&out.stdout).trim().to_string())
        .filter(|s| !s.is_empty())
}

pub(crate) fn stop_bundled_ollama_child(ollama_state: &OllamaChild) {
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
pub(crate) fn unload_session_ollama_models_on_exit() {
    let hub_base = std::env::var("NEURAL_JUNKIE_HUB_URL")
        .or_else(|_| std::env::var("VITE_NJ_HUB_URL"))
        .unwrap_or_else(|_| format!("http://127.0.0.1:{}", crate::DEFAULT_HUB_PORT));
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

pub(crate) fn spawn_bundled_ollama(app: &tauri::AppHandle, ollama_state: &OllamaChild) -> Option<PathBuf> {
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

#[derive(Debug, Serialize)]
pub struct OllamaRuntimeStatus {
    installed: bool,
    bundled: bool,
    running: bool,
    managed: bool,
    version: Option<String>,
}

#[tauri::command]
pub fn get_ollama_runtime_status(
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
pub fn start_bundled_ollama(
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
pub fn stop_bundled_ollama(
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
pub fn restart_bundled_ollama(
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

