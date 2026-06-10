// Command slack-oauth-relay is the public HTTPS Slack OAuth redirect relay for Neural Junkie.
//
// Slack requires HTTPS redirect URLs for distributed apps. This service receives the browser
// callback from Slack and forwards it to the user's local hub on loopback.
//
// Run locally:
//
//	go run ./cmd/slack-oauth-relay
//
// Deploy: scripts/deploy-slack-oauth-relay-aws.sh
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/camronwood/neural-junkie/internal/integrations/slack/relay"
)

func main() {
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(relay.HandleLambdaFunctionURL)
		return
	}
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":8080"
	}
	mux := relay.Mux()
	log.Printf("nj-slack-oauth-relay listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
