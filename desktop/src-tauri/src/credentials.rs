//! Machine-bound AES-GCM credential blob helpers for Tauri plugin-store.
//!
//! Phase 5 deferral: OS keychain (e.g. keyring crate) is not wired yet — keep
//! AES-GCM machine-bound encryption; migrate to keychain prefer+fallback later.

use aes_gcm::aead::{Aead, KeyInit};
use aes_gcm::{Aes256Gcm, Nonce};
use rand::RngCore;
use sha2::{Digest, Sha256};

fn machine_credential_key() -> [u8; 32] {
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
pub fn encrypt_credential_blob(plaintext: String) -> Result<String, String> {
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
pub fn decrypt_credential_blob(blob: String) -> Result<String, String> {
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
