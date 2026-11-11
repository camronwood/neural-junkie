//! Browser window / embedded browser commands (extraction from main.rs).

use tauri::{Emitter, Manager, WebviewUrl, WebviewWindowBuilder};
use tauri_plugin_opener::OpenerExt;

#[tauri::command]
pub async fn open_browser_window(url: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    // Never render arbitrary pages inside Neural Junkie.
    // In-app HTML preview belongs to the web-browser pack workbench only.
    // Always hand off to the OS default browser — http(s) only.
    let trimmed = url.trim();
    if trimmed.is_empty() {
        return Err("empty URL".to_string());
    }
    let lower = trimmed.to_ascii_lowercase();
    if !(lower.starts_with("http://") || lower.starts_with("https://")) {
        return Err("only http(s) URLs are allowed".to_string());
    }
    app_handle
        .opener()
        .open_url(trimmed, None::<&str>)
        .map_err(|e| format!("Failed to open URL in system browser: {}", e))?;
    Ok(())
}

#[tauri::command]
pub async fn close_browser_window(app_handle: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app_handle.get_webview_window("browser-popout") {
        window
            .close()
            .map_err(|e| format!("Failed to close browser window: {}", e))?;
    }
    Ok(())
}

#[tauri::command]
pub async fn capture_browser_screenshot(app_handle: tauri::AppHandle) -> Result<String, String> {
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
pub async fn navigate_browser(url: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    // This would communicate with the browser window to navigate to the URL
    // For now, just emit an event that the frontend can listen to
    app_handle
        .emit("browser-navigate", &url)
        .map_err(|e| format!("Failed to emit navigation event: {}", e))?;
    Ok(())
}

#[tauri::command]
pub async fn create_embedded_browser(
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
pub async fn update_browser_position(
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
pub async fn navigate_embedded_browser(
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
pub async fn destroy_embedded_browser(app_handle: tauri::AppHandle) -> Result<(), String> {
    if let Some(window) = app_handle.get_webview_window("embedded-browser") {
        window
            .close()
            .map_err(|e| format!("Failed to close embedded browser: {}", e))?;
        // Wait a moment to ensure window is fully closed
        tokio::time::sleep(tokio::time::Duration::from_millis(100)).await;
    }
    Ok(())
}

