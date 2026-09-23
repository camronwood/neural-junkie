package connectors

import "testing"

func TestApplyToSMSConfig(t *testing.T) {
	prof := &Profile{
		Type: TypeSMS,
		Config: map[string]string{
			"url":    "https://example.com/sms",
			"from":   "+1555",
			"format": "json",
		},
		Secret: "tok",
	}
	out := ApplyToSMSConfig(map[string]interface{}{"to": "+1"}, prof)
	if out["url"] != "https://example.com/sms" || out["from"] != "+1555" || out["format"] != "json" {
		t.Fatalf("out = %#v", out)
	}
	headers, _ := out["headers"].(map[string]interface{})
	if headers["Authorization"] != "Bearer tok" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestApplyToSMSConfigHTTPAuthWebhook(t *testing.T) {
	httpProf := &Profile{
		Type: TypeHTTPAuth,
		Config: map[string]string{
			"url":         "https://hooks.example/sms",
			"header_name": "X-Api-Key",
		},
		Secret: "sekret",
	}
	out := ApplyToSMSConfig(map[string]interface{}{"to": "+1", "body": "hi"}, httpProf)
	if out["url"] != "https://hooks.example/sms" {
		t.Fatalf("url = %#v", out["url"])
	}
	headers, _ := out["headers"].(map[string]interface{})
	if headers["X-Api-Key"] != "sekret" {
		t.Fatalf("headers = %#v", headers)
	}

	wh := &Profile{
		Type:   TypeWebhook,
		Config: map[string]string{"url": "https://hooks.example/wh"},
		Secret: "Bearer abc",
	}
	out2 := ApplyToSMSConfig(nil, wh)
	if out2["url"] != "https://hooks.example/wh" {
		t.Fatalf("webhook url = %#v", out2["url"])
	}
	h2, _ := out2["headers"].(map[string]interface{})
	if h2["Authorization"] != "Bearer abc" {
		t.Fatalf("webhook headers = %#v", h2)
	}
}

func TestApplyToEmailConfig(t *testing.T) {
	prof := &Profile{
		Type: TypeEmail,
		Config: map[string]string{
			"host":     "smtp.example.com",
			"port":     "587",
			"username": "u",
			"from":     "nj@example.com",
		},
		Secret: "pw",
	}
	out := ApplyToEmailConfig(map[string]interface{}{"to": "a@b.c"}, prof)
	if out["host"] != "smtp.example.com" || out["password"] != "pw" || out["from"] != "nj@example.com" {
		t.Fatalf("out = %#v", out)
	}
}
