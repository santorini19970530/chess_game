// CM3070 FP code
// analyzer_http.go - python analyzer/explain http client

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// analyzerBaseURL - returns the base URL for the analyzer
func analyzerBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

// analyzerRequestTimeout - returns the http timeout for analyzer calls
func analyzerRequestTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PY_ANALYSER_TIMEOUT_MS"))
	if raw == "" {
		return 2500 * time.Millisecond
	}
	timeoutMS, err := strconv.Atoi(raw)
	if err != nil || timeoutMS < 100 {
		return 2500 * time.Millisecond
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

// analyzerUserSafeError - maps an analyzer error to a user-safe message
func analyzerUserSafeError(err error) string {
	kind, _ := analyzerErrorDetails(err)
	return analyzerUserSafeErrorByKind(kind)
}

// analyzerUserSafeErrorByKind - maps an analyzer error kind to a user-safe message
func analyzerUserSafeErrorByKind(kind string) string {
	switch kind {
	case analysisErrorKindTimeout:
		return "Analysis timed out. Showing previous result."
	case analysisErrorKindUnavailable:
		return "Analysis service unavailable. Showing previous result."
	case analysisErrorKindBadStatus:
		return "Analysis service returned an invalid response."
	case analysisErrorKindBadJSON:
		return "Analysis response could not be processed."
	case analysisErrorKindNone:
		return ""
	default:
		return "Analysis temporarily unavailable. Showing previous result."
	}
}

// analyzerErrorDetails - builds structured analyzer error details for logs
func analyzerErrorDetails(err error) (string, int) {
	if err == nil {
		return analysisErrorKindNone, 0
	}
	var callErr *analyzerCallError
	if errors.As(err, &callErr) {
		kind := strings.TrimSpace(callErr.Kind)
		if kind == "" {
			kind = analysisErrorKindOther
		}
		return kind, callErr.HTTPStatus
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return analysisErrorKindTimeout, 0
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return analysisErrorKindTimeout, 0
		}
		return analysisErrorKindUnavailable, 0
	}
	return analysisErrorKindOther, 0
}

// emitAnalysisLog - emits analysis log
func emitAnalysisLog(entry analysisLogEvent) {
	if strings.TrimSpace(entry.ErrorKind) == "" {
		entry.ErrorKind = analysisErrorKindNone
	}
	if strings.TrimSpace(entry.TimestampUTC) == "" {
		entry.TimestampUTC = time.Now().UTC().Format(time.RFC3339Nano)
	}
	raw, err := json.Marshal(entry)
	if err != nil {
		log.Printf("warning: analysis log marshal failed: %v", err)
		return
	}
	log.Print(string(raw))
}

// analyzeByRequest - posts one /analyze request to the python service
func analyzeByRequest(reqPayload analyzerRequest) (*analyzerResponse, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("analyzer request marshal failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), analyzerRequestTimeout())
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, analyzerBaseURL()+"/analyze", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("analyzer request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pyAnalyzerHTTPClient.Do(req)
	if err != nil {
		kind, _ := analyzerErrorDetails(err)
		return nil, &analyzerCallError{
			Kind: kind,
			Err:  fmt.Errorf("analyzer request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("analyzer response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &analyzerCallError{
			Kind:       analysisErrorKindBadStatus,
			HTTPStatus: resp.StatusCode,
			Err:        fmt.Errorf("analyzer returned status=%d body=%s", resp.StatusCode, string(respBody)),
		}
	}

	var parsed analyzerResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &analyzerCallError{
			Kind: analysisErrorKindBadJSON,
			Err:  fmt.Errorf("analyzer response parse failed: %w", err),
		}
	}

	// printed for testing as requested.
	log.Printf("analyzer response: %s", string(respBody))
	return &parsed, nil
}

// explainByRequest - performs a POST to the Python /explain endpoint
func explainByRequest(reqPayload explainRequest) (*explainResponse, error) {
	body, err := json.Marshal(reqPayload)
	if err != nil {
		return nil, fmt.Errorf("explain request marshal failed: %w", err)
	}

	// use a slightly longer timeout than analysis because LLM generation can be slower.
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, analyzerBaseURL()+"/explain", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("explain request build failed: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := pyAnalyzerHTTPClient.Do(req)
	if err != nil {
		kind, _ := analyzerErrorDetails(err)
		return nil, &analyzerCallError{
			Kind: kind,
			Err:  fmt.Errorf("explain request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("explain response read failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &analyzerCallError{
			Kind:       analysisErrorKindBadStatus,
			HTTPStatus: resp.StatusCode,
			Err:        fmt.Errorf("explain returned status=%d body=%s", resp.StatusCode, string(respBody)),
		}
	}

	var parsed explainResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, &analyzerCallError{
			Kind: analysisErrorKindBadJSON,
			Err:  fmt.Errorf("explain response parse failed: %w", err),
		}
	}

	if len(respBody) > 240 {
		log.Printf("explain response: %s…", string(respBody[:240]))
	} else {
		log.Printf("explain response: %s", string(respBody))
	}
	return &parsed, nil
}
