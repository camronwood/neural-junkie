package devops

import "testing"

func TestIsValidK8sName(t *testing.T) {
	if !isValidK8sName("my-pod-1") {
		t.Fatal("expected valid name")
	}
	if isValidK8sName("-bad") {
		t.Fatal("expected invalid name starting with dash")
	}
	if isValidK8sName("--inject") {
		t.Fatal("expected flag-like name rejected")
	}
}

func TestIsAllowedK8sResource(t *testing.T) {
	if !isAllowedK8sResource("pods") {
		t.Fatal("pods should be allowed")
	}
	if isAllowedK8sResource("pods --all-namespaces") {
		t.Fatal("injection resource should be rejected")
	}
}

func TestIsSafeYamlPath(t *testing.T) {
	if !isSafeYamlPath("/tmp/manifest.yaml") {
		t.Fatal("expected absolute path ok")
	}
	if isSafeYamlPath("../etc/passwd") {
		t.Fatal("expected traversal rejected")
	}
}
