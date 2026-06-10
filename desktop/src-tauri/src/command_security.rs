/// Shell command allowlist/denylist for desktop execute_command (mirrors workspace MCP + protocol).
pub fn command_allowed(command: &str) -> bool {
    let cmd = normalize_command(command);
    if cmd.is_empty() {
        return false;
    }
    let lower = cmd.to_lowercase();

    let denied = [
        "rm -rf",
        "rm -r ",
        "sudo ",
        "curl ",
        "wget ",
        "| sh",
        "| bash",
        ">/dev/",
        "chmod ",
        "mkfs",
        "rm ",
        "rmdir",
        "del ",
        "kill",
        "killall",
        "shutdown",
        "reboot",
        "dd if=",
        "dd of=",
    ];
    for d in denied {
        if lower.contains(d) {
            return false;
        }
    }

    let allowed_prefixes = [
        "npm test",
        "npm run lint",
        "npm run test",
        "npm run build",
        "npm run typecheck",
        "npm exec -- tsc",
        "./node_modules/.bin/tsc",
        "go test",
        "go vet",
        "go list",
        "go version",
        "cargo test",
        "cargo check",
        "pytest",
        "python -m pytest",
        "ls",
        "pwd",
        "cat ",
        "head ",
        "tail ",
        "grep ",
        "git status",
        "git log",
        "git diff",
        "git show",
        "git branch",
        "make test",
    ];
    for p in allowed_prefixes {
        if lower.starts_with(p) {
            return true;
        }
    }
    false
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
}
