// CM3070 FP code
// diagram_fen_test.go - request_id passthrough on the diagram proxy

package handlers

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestFenFromImageByMultipart_ForwardsRequestID - python form field matches the caller id
func TestFenFromImageByMultipart_ForwardsRequestID(t *testing.T) {
	var gotID string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Errorf("parse: %v", err)
			http.Error(w, "bad multipart", http.StatusBadRequest)
			return
		}
		gotID = r.FormValue("request_id")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"status":"ok","fen":"8/8/8/8/8/8/8/8 w - - 0 1","game":"chess","request_id":"`+gotID+`"}`)
	}))
	defer srv.Close()
	t.Setenv("PY_ANALYSER_URL", srv.URL)

	out, err := fenFromImageByMultipart(
		context.Background(),
		[]byte("fake-image"),
		"board.png",
		"chess",
		"issue0068-c-test",
	)
	if err != nil {
		t.Fatalf("proxy: %v", err)
	}
	if gotID != "issue0068-c-test" {
		t.Fatalf("python request_id=%q", gotID)
	}
	if out.RequestID != "issue0068-c-test" {
		t.Fatalf("echo request_id=%q", out.RequestID)
	}
}
