/// Shell command allowlist/denylist for desktop execute_command (mirrors workspace MCP + protocol).
///
/// When `user_approved` is true (user clicked Run on an agent suggestion), only the hard
/// denylist applies — the allowlist is skipped because approval already happened in the UI.

pub fn command_hard_denied(command: &str) -> bool {
    let lower = normalize_command(command).to_lowercase();
    if lower.is_empty() {
        return true;
    }
    let denied = [
        "rm -rf", "rm -r ", "sudo ", "curl ", "wget ", "| sh", "| bash", ">/dev/", "chmod ",
        "mkfs", "rm ", "rmdir", "del ", "kill", "killall", "shutdown", "reboot", "dd if=",
        "dd of=",
    ];
    for d in denied {
        if lower.contains(d) {
            return true;
        }
    }
    false
}

pub fn command_allowed(command: &str) -> bool {
    command_allowed_with_approval(command, false)
}

pub fn command_allowed_with_approval(command: &str, user_approved: bool) -> bool {
    let cmd = normalize_command(command);
    if cmd.is_empty() {
        return false;
    }
    if command_hard_denied(&cmd) {
        return false;
    }
    if user_approved {
        return true;
    }
    let lower = cmd.to_lowercase();

    let allowed_prefixes = [
        "npm test",
        "npm run ",
        "npm exec ",
        "npx ",
        "yarn ",
        "pnpm ",
        "bun ",
        "cargo ",
        "tauri ",
        "go test",
        "go build",
        "go vet",
        "go list",
        "go version",
        "go run ",
        "pytest",
        "python -m pytest",
        "python -m ",
        "make ",
        "ls",
        "pwd",
        "cat ",
        "head ",
        "tail ",
        "grep ",
        "find ",
        "which ",
        "git status",
        "git log",
        "git diff",
        "git show",
        "git branch",
        "./node_modules/.bin/",
    ];
    for p in allowed_prefixes {
        if lower == p.trim() || lower.starts_with(p) {
            return true;
        }
    }
    // bare `ls` / `pwd` without trailing space
    matches!(lower.as_str(), "ls" | "pwd" | "whoami" | "date" | "uname")
}

fn normalize_command(cmd: &str) -> String {
    cmd.split_whitespace().collect::<Vec<_>>().join(" ")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn allows_go_test() {
        assert!(command_allowed("go test ./..."));
    }

    #[test]
    fn denies_rm_rf() {
        assert!(!command_allowed("rm -rf /"));
    }

    #[test]
    fn user_approved_allows_non_prefix_commands() {
        assert!(!command_allowed("custom-tool bootstrap"));
        assert!(command_allowed_with_approval("custom-tool bootstrap", true));
        assert!(!command_allowed_with_approval("rm -rf /", true));
        assert!(command_allowed("tauri dev"));
    }

    #[test]
    fn allows_npm_run_prefix() {
        assert!(command_allowed("npm run build"));
        assert!(command_allowed("npm run tauri"));
    }
}
