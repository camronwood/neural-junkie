package agent

import (
	"strings"
	"testing"
)

func TestExtractCodeFenceForPath_tailwind(t *testing.T) {
	t.Parallel()
	resp := "Here:\n```tsx\nexport default function App() {}\n```\n```js\nexport default { darkMode: 'class', content: ['./src/**/*'] }\n```"
	got := extractCodeFenceForPath(resp, "tailwind.config.js")
	if !strings.Contains(got, "darkMode") {
		t.Fatalf("want tailwind fence, got %q", got)
	}
}

func TestSynthesizeGoMainEdit_HelloWorld(t *testing.T) {
	t.Parallel()
	existing := "package main\n\nfunc main() {}\n"
	got, ok := synthesizeGoMainEdit("please implement HelloWorld in core/sample/main.go", existing)
	if !ok || !strings.Contains(got, "func HelloWorld") || !strings.Contains(got, "HelloWorld()") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeTailwindDarkMode(t *testing.T) {
	t.Parallel()
	existing := "export default {\n  content: ['./index.html'],\n}\n"
	got, ok := synthesizeTailwindDarkMode(existing)
	if !ok || !strings.Contains(got, "darkMode") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeTailwindDarkMode_CJS(t *testing.T) {
	t.Parallel()
	existing := "module.exports = {\n  theme: { extend: {} },\n}\n"
	got, ok := synthesizeTailwindDarkMode(existing)
	if !ok || !strings.Contains(got, "darkMode") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}

func TestSynthesizeGoMainEdit_PrintVersion(t *testing.T) {
	t.Parallel()
	existing := "package main\n\nfunc main() {}\n"
	got, ok := synthesizeGoMainEdit("implement a PrintVersion helper in core/sample/main.go", existing)
	if !ok || !strings.Contains(got, "func PrintVersion") {
		t.Fatalf("got ok=%v body=%q", ok, got)
	}
}
