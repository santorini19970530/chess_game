package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AIClient calls the Python service /history, /policy, and /value endpoints.
type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewAIClient builds a client using existing analyzer config defaults.
func NewAIClient() *AIClient {
	return &AIClient{
		baseURL:    analyzerBaseURL(),
		httpClient: pyAnalyzerHTTPClient,
	}
}

// AICommonRequest is the shared request envelope for three-agent endpoints.
type AICommonRequest struct {
	RequestID   string   `json:"request_id"`
	GameID      string   `json:"game_id,omitempty"`
	GameType    string   `json:"game_type"`
	Variant     string   `json:"variant,omitempty"`
	FEN         string   `json:"fen"`
	Color       string   `json:"color"`
	MoveNumber  int      `json:"move_number,omitempty"`
	MoveHistory []string `json:"move_history,omitempty"`
}

// AIPolicyRequest extends AICommonRequest for policy-specific options.
type AIPolicyRequest struct {
	AICommonRequest
	TopK int `json:"top_k,omitempty"`
}

type aiErrorResponse struct {
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	ErrorKind string `json:"error_kind"`
	Message   string `json:"message"`
}

const (
	aiClientErrorKindTimeout     = "timeout"
	aiClientErrorKindUnavailable = "unavailable"
	aiClientErrorKindBadStatus   = "bad_status"
	aiClientErrorKindBadJSON     = "bad_json"
	aiClientErrorKindRequest     = "request_error"
	aiClientErrorKindResponse    = "response_error"
	aiClientErrorKindOther       = "other"
)

// AIClientError standardizes failure kinds from AI endpoint calls.
type AIClientError struct {
	Kind       string
	HTTPStatus int
	Err        error
}

func (e *AIClientError) Error() string {
	if e == nil || e.Err == nil {
		return "ai client error"
	}
	return e.Err.Error()
}

func (e *AIClientError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

type AIHistoryResponse struct {
	RequestID string                 `json:"request_id"`
	Status    string                 `json:"status"`
	Source    string                 `json:"source"`
	Phase     string                 `json:"phase"`
	Features  map[string]interface{} `json:"features"`
	Tags      []string               `json:"tags"`
	LatencyMS int                    `json:"latency_ms"`
}

type AIPolicyCandidate struct {
	Rank    int     `json:"rank"`
	UCI     string  `json:"uci"`
	SAN     string  `json:"san"`
	ScoreCP int     `json:"score_cp"`
	Prob    float64 `json:"prob"`
}

type AIPolicyResponse struct {
	RequestID   string              `json:"request_id"`
	Status      string              `json:"status"`
	Source      string              `json:"source"`
	BestMoveUCI string              `json:"best_move_uci"`
	Candidates  []AIPolicyCandidate `json:"candidates"`
	LatencyMS   int                 `json:"latency_ms"`
}

type AIValueResponse struct {
	RequestID      string  `json:"request_id"`
	Status         string  `json:"status"`
	Source         string  `json:"source"`
	ScoreCP        int     `json:"score_cp"`
	MateIn         int     `json:"mate_in"`
	Value          float64 `json:"value"`
	WinChanceWhite float64 `json:"win_chance_white"`
	WinChanceBlack float64 `json:"win_chance_black"`
	LatencyMS      int     `json:"latency_ms"`
}

func (c *AIClient) History(req AICommonRequest) (*AIHistoryResponse, error) {
	var out AIHistoryResponse
	if err := c.doJSONPost("/history", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AIClient) Policy(req AIPolicyRequest) (*AIPolicyResponse, error) {
	var out AIPolicyResponse
	if err := c.doJSONPost("/policy", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AIClient) Value(req AICommonRequest) (*AIValueResponse, error) {
	var out AIValueResponse
	if err := c.doJSONPost("/value", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// doJSONPost is a shared helper for JSON POST calls to AI endpoints.
func (c *AIClient) doJSONPost(path string, payload any, out any) error {
	baseURL := strings.TrimRight(c.baseURL, "/")
	client := c.httpClient
	if client == nil {
		client = pyAnalyzerHTTPClient
	}
	return doJSONPost(client, baseURL+path, analyzerRequestTimeout(), payload, out)
}

// doJSONPost executes a JSON HTTP POST and decodes JSON response into out.
func doJSONPost(httpClient *http.Client, endpoint string, timeout time.Duration, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return &AIClientError{
			Kind: aiClientErrorKindRequest,
			Err:  fmt.Errorf("ai client request marshal failed: %w", err),
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return &AIClientError{
			Kind: aiClientErrorKindRequest,
			Err:  fmt.Errorf("ai client request build failed: %w", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")

	if httpClient == nil {
		httpClient = pyAnalyzerHTTPClient
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return &AIClientError{
			Kind: mapAIClientTransportErrorKind(err),
			Err:  fmt.Errorf("ai client request failed: %w", err),
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &AIClientError{
			Kind: aiClientErrorKindResponse,
			Err:  fmt.Errorf("ai client response read failed: %w", err),
		}
	}

	if resp.StatusCode != http.StatusOK {
		var parsedErr aiErrorResponse
		if err := json.Unmarshal(respBody, &parsedErr); err == nil && parsedErr.Message != "" {
			return &AIClientError{
				Kind:       aiClientErrorKindBadStatus,
				HTTPStatus: resp.StatusCode,
				Err:        fmt.Errorf("ai client returned status=%d error=%s", resp.StatusCode, parsedErr.Message),
			}
		}
		return &AIClientError{
			Kind:       aiClientErrorKindBadStatus,
			HTTPStatus: resp.StatusCode,
			Err:        fmt.Errorf("ai client returned status=%d body=%s", resp.StatusCode, string(respBody)),
		}
	}

	if err := json.Unmarshal(respBody, out); err != nil {
		return &AIClientError{
			Kind: aiClientErrorKindBadJSON,
			Err:  fmt.Errorf("ai client response parse failed: %w", err),
		}
	}
	return nil
}

func mapAIClientTransportErrorKind(err error) string {
	if err == nil {
		return aiClientErrorKindOther
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return aiClientErrorKindTimeout
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if errors.Is(urlErr.Err, context.DeadlineExceeded) {
			return aiClientErrorKindTimeout
		}
		return aiClientErrorKindUnavailable
	}
	return aiClientErrorKindOther
}
