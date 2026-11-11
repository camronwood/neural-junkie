//! Pack directory / zip filesystem commands (extraction from main.rs).

use serde::{Deserialize, Serialize};
use tauri::Manager;
use tauri_plugin_dialog::DialogExt;

const MAX_PACK_ZIP_BYTES: usize = 10 << 20; // 10 MiB — matches hub internal/packs

#[derive(Debug, Serialize, Deserialize)]
pub struct PackScaffoldRequest {
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

#[tauri::command]
pub fn register_pack_path(path: String, app_handle: tauri::AppHandle) -> Result<(), String> {
    let trimmed = path.trim();
    if trimmed.is_empty() {
        return Err("empty path".into());
    }
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::allowlist_insert(allowlist.inner(), trimmed)?;
    Ok(())
}

/// Pick a pack directory via native folder dialog.
#[tauri::command]
pub fn pick_pack_directory(
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
        let allowlist = app_handle.state::<crate::PackPathAllowlist>();
        crate::allowlist_insert(allowlist.inner(), path)?;
    }
    Ok(picked)
}

/// Create scaffold pack.yaml and starter assets in output_dir.
#[tauri::command]
pub fn write_pack_scaffold(
    req: PackScaffoldRequest,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::path_allowed(&allowed_roots, allowlist.inner(), &req.output_dir)?;
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
pub fn read_pack_yaml_from_dir(
    absolute_dir: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
    let dir = absolute_dir.trim();
    if dir.is_empty() {
        return Err("empty directory".into());
    }
    let path = format!("{}/pack.yaml", dir);
    std::fs::read_to_string(&path).map_err(|e| format!("read {}: {}", path, e))
}

/// Write pack.yaml to an absolute pack directory.
#[tauri::command]
pub fn write_pack_yaml_to_dir(
    absolute_dir: String,
    yaml: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<(), String> {
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
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
pub fn zip_pack_directory(
    absolute_dir: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::path_allowed(&allowed_roots, allowlist.inner(), &absolute_dir)?;
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

pub fn walkdir_for_pack(root: &std::path::Path) -> Result<Vec<std::path::PathBuf>, String> {
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
pub fn read_pack_zip_base64(
    absolute_path: String,
    allowed_roots: Vec<String>,
    app_handle: tauri::AppHandle,
) -> Result<String, String> {
    let allowlist = app_handle.state::<crate::PackPathAllowlist>();
    crate::path_allowed(&allowed_roots, allowlist.inner(), &absolute_path)?;
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


