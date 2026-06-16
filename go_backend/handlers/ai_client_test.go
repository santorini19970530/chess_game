package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestAIClient(baseURL string) *AIClient {
	return &AIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 0,
		},
	}
}

func baseAIRequest() AICommonRequest {
	return AICommonRequest{
		RequestID:  "test-req-1",
		GameID:     "game-1",
		GameType:   "chess",
		Variant:    "chess",
		FEN:        "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
		Color:      "white",
		MoveNumber: 1,
	}
}

func TestAIClientSuccessForEachEndpoint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/history":
			_, _ = w.Write([]byte(`{"request_id":"test-req-1","status":"ok","source":"rule_based_v1","phase":"opening","features":{"is_check":false},"tags":["balanced"],"latency_ms":1}`))
		case "/policy":
			_, _ = w.Write([]byte(`{"request_id":"test-req-1","status":"ok","source":"heuristic","best_move_uci":"e2e4","candidates":[{"rank":1,"uci":"e2e4","san":"e4","score_cp":20,"prob":0.6}],"latency_ms":2}`))
		case "/value":
			_, _ = w.Write([]byte(`{"request_id":"test-req-1","status":"ok","source":"heuristic","score_cp":20,"mate_in":0,"value":0.2,"win_chance_white":0.6,"win_chance_black":0.4,"latency_ms":1}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := newTestAIClient(srv.URL)
	req := baseAIRequest()

	history, err := client.History(req)
	if err != nil {
		t.Fatalf("history call failed: %v", err)
	}
	if history.Status != "ok" {
		t.Fatalf("expected history status ok, got %q", history.Status)
	}

	policy, err := client.Policy(AIPolicyRequest{AICommonRequest: req, TopK: 3})
	if err != nil {
		t.Fatalf("policy call failed: %v", err)
	}
	if policy.BestMoveUCI == "" {
		t.Fatalf("expected best move uci in policy response")
	}

	value, err := client.Value(req)
	if err != nil {
		t.Fatalf("value call failed: %v", err)
	}
	if value.Status != "ok" {
		t.Fatalf("expected value status ok, got %q", value.Status)
	}
}

func TestAIClientTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(250 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"test-req-1","status":"ok","source":"rule_based_v1","phase":"opening","features":{"is_check":false},"tags":["balanced"],"latency_ms":1}`))
	}))
	defer srv.Close()

	t.Setenv("PY_ANALYSER_TIMEOUT_MS", "100")
	client := newTestAIClient(srv.URL)
	_, err := client.History(baseAIRequest())
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	var clientErr *AIClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected AIClientError, got %T", err)
	}
	if clientErr.Kind != aiClientErrorKindTimeout {
		t.Fatalf("expected timeout kind, got %q", clientErr.Kind)
	}
}

func TestAIClientNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"error","error_kind":"unavailable","message":"service down"}`))
	}))
	defer srv.Close()

	client := newTestAIClient(srv.URL)
	_, err := client.Policy(AIPolicyRequest{AICommonRequest: baseAIRequest(), TopK: 3})
	if err == nil {
		t.Fatalf("expected non-200 error, got nil")
	}
	var clientErr *AIClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected AIClientError, got %T", err)
	}
	if clientErr.Kind != aiClientErrorKindBadStatus {
		t.Fatalf("expected bad_status kind, got %q", clientErr.Kind)
	}
	if clientErr.HTTPStatus != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", clientErr.HTTPStatus)
	}
}

func TestAIClientMalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"request_id":"oops"`))
	}))
	defer srv.Close()

	client := newTestAIClient(srv.URL)
	_, err := client.Value(baseAIRequest())
	if err == nil {
		t.Fatalf("expected malformed json error, got nil")
	}
	var clientErr *AIClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected AIClientError, got %T", err)
	}
	if clientErr.Kind != aiClientErrorKindBadJSON {
		t.Fatalf("expected bad_json kind, got %q", clientErr.Kind)
	}
}

func TestAIClientMissingRequiredFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// request_id is intentionally missing.
		_, _ = w.Write([]byte(`{"status":"ok","source":"rule_based_v1","phase":"opening","features":{"is_check":false},"tags":["balanced"],"latency_ms":1}`))
	}))
	defer srv.Close()

	client := newTestAIClient(srv.URL)
	_, err := client.History(baseAIRequest())
	if err == nil {
		t.Fatalf("expected missing field validation error, got nil")
	}
	var clientErr *AIClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected AIClientError, got %T", err)
	}
	if clientErr.Kind != aiClientErrorKindBadPayload {
		t.Fatalf("expected bad_payload kind, got %q", clientErr.Kind)
	}
	if !strings.Contains(err.Error(), "request_id") {
		t.Fatalf("expected missing request_id message, got %v", err)
	}
}

func TestAIClientNetworkUnavailableMapping(t *testing.T) {
	// Port 1 should be closed and return a connection-refused style network error.
	client := newTestAIClient("http://127.0.0.1:1")
	_, err := client.History(baseAIRequest())
	if err == nil {
		t.Fatalf("expected unavailable network error, got nil")
	}
	var clientErr *AIClientError
	if !errors.As(err, &clientErr) {
		t.Fatalf("expected AIClientError, got %T", err)
	}
	if clientErr.Kind != aiClientErrorKindUnavailable {
		t.Fatalf("expected unavailable kind, got %q", clientErr.Kind)
	}
}
