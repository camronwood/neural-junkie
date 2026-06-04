package phoeniximport

import "fmt"

type environmentProfile struct {
	APIBase      string
	Auth0Domain  string
	Audience     string
	DefaultEnv   string
}

func profileForEnvironment(env string) (environmentProfile, error) {
	switch normalizeEnvironment(env) {
	case "dev":
		return environmentProfile{
			APIBase:     "https://8d3hbybxc7.execute-api.us-east-2.amazonaws.com/api",
			Auth0Domain: "dev-zazkmky7c1v5de5q.us.auth0.com",
			Audience:    "https://api.dev.brightestbio.com",
		}, nil
	case "prod":
		return environmentProfile{}, fmt.Errorf("prod TIM API is not configured yet; use dev or staging")
	case "staging", "":
		return environmentProfile{
			APIBase:     "https://o6l9ljpw8f.execute-api.us-east-2.amazonaws.com/api",
			Auth0Domain: "brightestbio.us.auth0.com",
			Audience:    "https://api.staging.brightestbio.com",
		}, nil
	default:
		return environmentProfile{}, fmt.Errorf("unknown environment %q; use staging, dev, or prod", env)
	}
}

func normalizeEnvironment(env string) string {
	switch env {
	case "staging", "dev", "prod":
		return env
	default:
		return "staging"
	}
}
