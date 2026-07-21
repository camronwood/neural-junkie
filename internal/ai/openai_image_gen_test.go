package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type imageRoundTripperFunc func(*http.Request) (*http.Response, error)

func (f imageRoundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type failingImageResponseBody struct{}

func (failingImageResponseBody) Read([]byte) (int, error) {
	return 0, errors.New("stream reset")
}

func (failingImageResponseBody) Close() error {
	return nil
}

func TestGenerateImageReportsResponseReadError(t *testing.T) {
	p := NewOpenAICompatProvider("http://example.test/v1", "", "image-model", nil)
	p.SetHTTPClient(&http.Client{
		Transport: imageRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       failingImageResponseBody{},
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, _, err := p.GenerateImage(context.Background(), "a blue circle", "")
	if err == nil || !strings.Contains(err.Error(), "read images API response: stream reset") {
		t.Fatalf("expected response read error, got %v", err)
	}
}

func TestGenerateImageReportsEmptyResponse(t *testing.T) {
	p := NewOpenAICompatProvider("http://example.test/v1", "", "image-model", nil)
	p.SetHTTPClient(&http.Client{
		Transport: imageRoundTripperFunc(func(*http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(" \n")),
				Header:     make(http.Header),
			}, nil
		}),
	})

	_, _, err := p.GenerateImage(context.Background(), "a blue circle", "")
	if err == nil || err.Error() != "images API returned an empty response" {
		t.Fatalf("expected empty response error, got %v", err)
	}
}
