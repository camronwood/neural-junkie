package relay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// HandleLambdaFunctionURL serves relay routes from an AWS Lambda Function URL event.
func HandleLambdaFunctionURL(ctx context.Context, req events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {
	_ = ctx
	path := req.RawPath
	if path == "" {
		path = req.RequestContext.HTTP.Path
	}
	httpReq, err := http.NewRequest(strings.ToUpper(req.RequestContext.HTTP.Method), path, nil)
	if err != nil {
		return events.LambdaFunctionURLResponse{StatusCode: http.StatusInternalServerError, Body: err.Error()}, nil
	}
	if httpReq.URL != nil && req.RawQueryString != "" {
		httpReq.URL.RawQuery = req.RawQueryString
	}
	rec := httptest.NewRecorder()
	Mux().ServeHTTP(rec, httpReq)
	headers := map[string]string{}
	for k, vals := range rec.Header() {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: rec.Code,
		Headers:    headers,
		Body:       rec.Body.String(),
	}, nil
}
