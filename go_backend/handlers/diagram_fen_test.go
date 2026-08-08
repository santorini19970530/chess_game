// CM3070 FP code
// diagram_fen_test.go - api checks for diagram→fen go proxy

package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestPostAPIDiagramFen_ProxiesOK - checks /api/diagram/fen forwards to python and returns fen
func TestPostAPIDiagramFen_ProxiesOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fen_from_image" {
			t.Errorf("path=%s want /fen_from_image", r.URL.Path)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		if r.FormValue("game") != "chess" {
			t.Errorf("game=%q want chess", r.FormValue("game"))
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			t.Fatalf("image: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "fake-png" {
			t.Fatalf("image bytes=%q", data)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status": "ok",
			"fen":    diagramChessFEN,
			"game":   "chess",
			"limits_note": "Chess strongest; Xiangqi OK; Shogi weaker / no pieces-in-hand",
		})
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)
	t.Setenv("PY_DIAGRAM_TIMEOUT_MS", "2000")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", "board.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("fake-png")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("game", "chess"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	h := NewHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/diagram/fen", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.postAPIDiagramFen(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var payload diagramFenResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Status != "ok" || payload.FEN != diagramChessFEN {
		t.Fatalf("payload=%+v", payload)
	}
}

// TestPostAPIDiagramFen_MissingImage - checks proxy rejects missing file
func TestPostAPIDiagramFen_MissingImage(t *testing.T) {
	h := NewHandler()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("game", "chess")
	_ = writer.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/diagram/fen", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.postAPIDiagramFen(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
}

// TestPostAPIDiagramFen_UpstreamError - checks proxy surfaces python validation errors
func TestPostAPIDiagramFen_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"status":"error","message":"No board detected in image","error_kind":"recognition"}`))
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)
	t.Setenv("PY_DIAGRAM_TIMEOUT_MS", "2000")

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, _ := writer.CreateFormFile("image", "board.png")
	_, _ = part.Write([]byte("x"))
	_ = writer.WriteField("game", "chess")
	_ = writer.Close()

	h := NewHandler()
	req := httptest.NewRequest(http.MethodPost, "/api/diagram/fen", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	h.postAPIDiagramFen(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want 400 body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "No board detected") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}
