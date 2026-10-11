use std::path::{Path, PathBuf};

/// Resolve a path to an absolute form. Uses canonicalize when the path exists;
/// otherwise resolves the parent and joins the final component.
fn resolve_abs(path: &str) -> Result<PathBuf, String> {
    let trimmed = path.trim();
    if trimmed.is_empty() {
        return Err("empty path".into());
    }
    let p = Path::new(trimmed);
    if p.exists() {
        return p
            .canonicalize()
            .map_err(|e| format!("invalid path {}: {}", trimmed, e));
    }
    if let Some(parent) = p.parent() {
        if parent.as_os_str().is_empty() || parent.exists() {
            let parent_abs = if parent.as_os_str().is_empty() {
                std::env::current_dir().map_err(|e| e.to_string())?
            } else {
                parent
                    .canonicalize()
                    .map_err(|e| format!("invalid parent for {}: {}", trimmed, e))?
            };
            let name = p
                .file_name()
                .ok_or_else(|| format!("invalid path: {}", trimmed))?;
            return Ok(parent_abs.join(name));
        }
    }
    Err(format!("path not found: {}", trimmed))
}

/// Separator-aware containment check (mirrors Go pathutil.AssertWithinRootAbs).
fn assert_within_root_abs(root_abs: &Path, cand_abs: &Path) -> Result<(), String> {
    if cand_abs == root_abs {
        return Ok(());
    }
    let root_str = root_abs.to_string_lossy();
    let cand_str = cand_abs.to_string_lossy();
    let sep = std::path::MAIN_SEPARATOR;
    let prefix = format!("{}{}", root_str, sep);
    if cand_str.starts_with(&prefix) {
        Ok(())
    } else {
        Err("path outside allowed root".into())
    }
}

/// Returns canonical absolute path if `candidate` resolves inside any of `roots`.
pub fn within_any_root(roots: &[String], candidate: &str) -> Result<PathBuf, String> {
    if roots.is_empty() {
        return Err("no allowed workspace roots configured".into());
    }
    let cand_abs = resolve_abs(candidate)?;
    for root in roots {
        let root_trim = root.trim();
        if root_trim.is_empty() {
            continue;
        }
        let root_abs = resolve_abs(root_trim)?;
        if assert_within_root_abs(&root_abs, &cand_abs).is_ok() {
            return Ok(cand_abs);
        }
    }
    Err(format!("path outside workspace: {}", candidate))
}

/// Validate `working_dir` when provided; empty roots skips validation (legacy dev).
pub fn validate_working_dir(roots: &[String], working_dir: Option<&str>) -> Result<(), String> {
    match working_dir {
        None | Some("") => Ok(()),
        Some(dir) => {
            within_any_root(roots, dir)?;
            Ok(())
        }
    }
}

/// Canonical absolute path if `candidate` resolves inside any allowlisted root.
/// Rejects `..` traversal that would escape the allowlisted directory.
pub fn within_any_allowlist(allowlist: &[String], candidate: &str) -> Result<PathBuf, String> {
    if allowlist.is_empty() {
        return Err("path not on pack allowlist".into());
    }
    within_any_root(allowlist, candidate).map_err(|_| format!("path outside allowlist: {}", candidate))
}

/// Resolve and canonicalize a path for allowlist insertion (picker / register).
pub fn canonicalize_for_allowlist(path: &str) -> Result<PathBuf, String> {
    let abs = resolve_abs(path)?;
    let s = abs.to_string_lossy();
    if s.contains("..") {
        return Err("path must not contain .. after resolution".into());
    }
    Ok(abs)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::env;
    use std::fs;

    #[test]
    fn rejects_prefix_sibling() {
        let tmp = env::temp_dir();
        let root = tmp.join("nj_path_root");
        let sibling = tmp.join("nj_path_root_evil");
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&sibling);
        fs::create_dir_all(&root).unwrap();
        fs::create_dir_all(&sibling).unwrap();
        let inside = root.join("ok.txt");
        fs::write(&inside, "x").unwrap();
        let roots = vec![root.to_string_lossy().into_owned()];
        assert!(within_any_root(&roots, inside.to_str().unwrap()).is_ok());
        let evil = sibling.join("oops.txt");
        fs::write(&evil, "y").unwrap();
        assert!(within_any_root(&roots, evil.to_str().unwrap()).is_err());
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&sibling);
    }

    #[test]
    fn allowlist_rejects_dotdot_escape() {
        let tmp = env::temp_dir();
        let root = tmp.join("nj_pack_allow_root");
        let sibling = tmp.join("nj_pack_allow_evil");
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&sibling);
        fs::create_dir_all(&root).unwrap();
        fs::create_dir_all(&sibling).unwrap();
        fs::write(sibling.join("oops.txt"), "y").unwrap();
        let allow = vec![root.to_string_lossy().into_owned()];
        let escape = root.join("../../../").join(sibling.file_name().unwrap()).join("oops.txt");
        let escape_str = escape.to_string_lossy().into_owned();
        assert!(within_any_allowlist(&allow, &escape_str).is_err());
        let inside = root.join("pack.yaml");
        fs::write(&inside, "x").unwrap();
        assert!(within_any_allowlist(&allow, inside.to_str().unwrap()).is_ok());
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&sibling);
    }

    #[test]
    fn rejects_symlink_escape() {
        let tmp = env::temp_dir();
        let root = tmp.join("nj_path_symlink_root");
        let outside = tmp.join("nj_path_symlink_outside");
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&outside);
        fs::create_dir_all(&root).unwrap();
        fs::create_dir_all(&outside).unwrap();
        fs::write(outside.join("secret.txt"), "x").unwrap();
        let link = root.join("escape");
        #[cfg(unix)]
        {
            use std::os::unix::fs::symlink;
            if symlink(&outside, &link).is_err() {
                return;
            }
            let roots = vec![root.to_string_lossy().into_owned()];
            let candidate = link.join("secret.txt");
            assert!(
                within_any_root(&roots, candidate.to_str().unwrap()).is_err(),
                "symlink escape should be rejected"
            );
        }
        let _ = fs::remove_dir_all(&root);
        let _ = fs::remove_dir_all(&outside);
    }
}
