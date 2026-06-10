#!/usr/bin/env bash
# Deploy nj-slack-oauth-relay to AWS Lambda (Function URL, HTTPS).
#
# Prerequisites:
#   aws sso login --profile AdministratorAccess-566982197870
#   go 1.25+
#
# Usage:
#   AWS_PROFILE=AdministratorAccess-566982197870 ./scripts/deploy-slack-oauth-relay-aws.sh
#
# Optional env:
#   AWS_REGION=us-east-2
#   NJ_SLACK_RELAY_FUNCTION=nj-slack-oauth-relay
#   NJ_SLACK_RELAY_DOMAIN=slack.oauth.neural-junkie.dev  # custom domain (manual ACM + API mapping)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REGION="${AWS_REGION:-us-east-2}"
FUNCTION="${NJ_SLACK_RELAY_FUNCTION:-nj-slack-oauth-relay}"
ROLE_NAME="${NJ_SLACK_RELAY_ROLE:-nj-slack-oauth-relay-lambda}"
ZIP="${ROOT}/bin/slack-oauth-relay-lambda.zip"
BUILD_DIR="${ROOT}/bin/slack-oauth-relay-lambda-build"

echo "==> Building Lambda binary (linux/amd64)"
rm -rf "$BUILD_DIR" "$ZIP"
mkdir -p "$BUILD_DIR"
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o "$BUILD_DIR/bootstrap" "$ROOT/cmd/slack-oauth-relay"
chmod +x "$BUILD_DIR/bootstrap"
(
  cd "$BUILD_DIR"
  zip -q "$ZIP" bootstrap
)

ACCOUNT_ID="$(aws sts get-caller-identity --query Account --output text)"
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
  echo "==> Creating IAM role ${ROLE_NAME}"
  TRUST='{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}'
  aws iam create-role --role-name "$ROLE_NAME" --assume-role-policy-document "$TRUST" >/dev/null
  aws iam attach-role-policy --role-name "$ROLE_NAME" \
    --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole >/dev/null
  echo "Waiting for IAM role propagation..."
  sleep 10
fi

if aws lambda get-function --function-name "$FUNCTION" --region "$REGION" >/dev/null 2>&1; then
  echo "==> Updating Lambda ${FUNCTION}"
  aws lambda update-function-code \
    --function-name "$FUNCTION" \
    --region "$REGION" \
    --zip-file "fileb://${ZIP}" >/dev/null
else
  echo "==> Creating Lambda ${FUNCTION}"
  aws lambda create-function \
    --function-name "$FUNCTION" \
    --region "$REGION" \
    --runtime provided.al2023 \
    --role "$ROLE_ARN" \
    --handler bootstrap \
    --zip-file "fileb://${ZIP}" \
    --timeout 10 \
    --memory-size 128 >/dev/null
fi

echo "==> Ensuring Function URL (public HTTPS)"
if ! aws lambda get-function-url-config --function-name "$FUNCTION" --region "$REGION" >/dev/null 2>&1; then
  aws lambda create-function-url-config \
    --function-name "$FUNCTION" \
    --region "$REGION" \
    --auth-type NONE \
    --cors '{"AllowOrigins":["*"],"AllowMethods":["GET"],"MaxAge":86400}' >/dev/null
  aws lambda add-permission \
    --function-name "$FUNCTION" \
    --region "$REGION" \
    --statement-id FunctionURLAllowPublicAccess \
    --action lambda:InvokeFunctionUrl \
    --principal "*" \
    --function-url-auth-type NONE >/dev/null 2>&1 || true
fi

FUNC_URL="$(aws lambda get-function-url-config --function-name "$FUNCTION" --region "$REGION" --query FunctionUrl --output text)"
FUNC_URL="${FUNC_URL%/}"

echo ""
echo "Relay deployed."
echo "  Lambda Function URL: ${FUNC_URL}"
echo ""
echo "Slack app OAuth redirect URLs (OAuth & Permissions):"
echo "  ${FUNC_URL}/api/slack/oauth/callback"
echo "  ${FUNC_URL}/api/slack/oauth/user-dm/callback"
echo ""
echo "Set vendor relay base (CI secret or local vendor/oauth.json):"
echo "  oauth_relay_base: ${FUNC_URL}"
echo "  export SLACK_VENDOR_OAUTH_RELAY_BASE=${FUNC_URL}"
echo "  export NEURAL_JUNKIE_SLACK_OAUTH_RELAY_BASE=${FUNC_URL}"
echo ""
echo "Optional: map a custom domain (e.g. slack.oauth.neural-junkie.dev) via ACM + CloudFront/API Gateway in front of the Function URL."
