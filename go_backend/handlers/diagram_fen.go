// CM3070 FP code
// diagram_fen.go - proxies board-diagram uploads to python /fen_from_image

package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const maxDiagramUploadBytes = 12 << 20

// diagramFenResponse - json returned to the frontend after a successful proxy call
type diagramFenResponse struct {
	Status              string `json:"status"`
	FEN                 string `json:"fen"`
	Game                string `json:"game"`
	BoardIsFlipped      *bool  `json:"board_is_flipped,omitempty"`
	ImageRotationAngle  *int   `json:"image_rotation_angle,omitempty"`
	LimitsNote          string `json:"limits_note,omitempty"`
	RequestID           string `json:"request_id,omitempty"`
}

// diagramTimeout - returns the http timeout for diagram recognition (slow cnn)
func diagramTimeout() time.Duration {
	raw := strings.TrimSpace(os.Getenv("PY_DIAGRAM_TIMEOUT_MS"))
	if raw == "" {
		return 120 * time.Second
	}
	timeoutMS, err := strconv.Atoi(raw)
	if err != nil || timeoutMS < 1000 {
		return 120 * time.Second
	}
	return time.Duration(timeoutMS) * time.Millisecond
}

// fenFromImageByMultipart - posts multipart image+game to python /fen_from_image
func fenFromImageByMultipart(ctx context.Context, imageBytes []byte, filename, game string) (diagramFenResponse, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("image", filename)
	if err != nil {
		return diagramFenResponse{}, err
	}
	if _, err := part.Write(imageBytes); err != nil {
		return diagramFenResponse{}, err
	}
	if err := writer.WriteField("game", game); err != nil {
		return diagramFenResponse{}, err
	}
	if err := writer.Close(); err != nil {
		return diagramFenResponse{}, err
	}

	url := strings.TrimRight(analyzerBaseURL(), "/") + "/fen_from_image"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return diagramFenResponse{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: diagramTimeout()}
	resp, err := client.Do(req)
	if err != nil {
		return diagramFenResponse{}, fmt.Errorf("diagram service unavailable: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return diagramFenResponse{}, fmt.Errorf("diagram service read error: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errPayload struct {
			Message string `json:"message"`
			Status  string `json:"status"`
		}
		_ = json.Unmarshal(respBody, &errPayload)
		msg := strings.TrimSpace(errPayload.Message)
		if msg == "" {
			msg = strings.TrimSpace(string(respBody))
		}
		if msg == "" {
			msg = fmt.Sprintf("diagram service status %d", resp.StatusCode)
		}
		return diagramFenResponse{}, &diagramProxyError{Status: resp.StatusCode, Message: msg}
	}

	var out diagramFenResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return diagramFenResponse{}, fmt.Errorf("diagram service bad json: %w", err)
	}
	if strings.TrimSpace(out.FEN) == "" {
		return diagramFenResponse{}, fmt.Errorf("diagram service returned empty fen")
	}
	if out.Status == "" {
		out.Status = "ok"
	}
	return out, nil
}

// diagramProxyError - carries upstream http status for the diagram proxy
type diagramProxyError struct {
	Status  int
	Message string
}

// Error - returns the proxy error message
func (e *diagramProxyError) Error() string {
	return e.Message
}

// APIDiagramFen - exported route entry for /api/diagram/fen
func (h *Handler) APIDiagramFen(w http.ResponseWriter, r *http.Request) {
	h.postAPIDiagramFen(w, r)
}

// postAPIDiagramFen - accepts a board image and returns fen from the python vision model
func (h *Handler) postAPIDiagramFen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSONError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	if err := r.ParseMultipartForm(maxDiagramUploadBytes); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid multipart diagram upload")
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		file, header, err = r.FormFile("file")
	}
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, `Missing multipart file field: "image"`)
		return
	}
	defer file.Close()

	imageBytes, err := io.ReadAll(io.LimitReader(file, maxDiagramUploadBytes+1))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Failed to read uploaded image")
		return
	}
	if len(imageBytes) == 0 {
		writeJSONError(w, http.StatusBadRequest, "Empty image upload")
		return
	}
	if len(imageBytes) > maxDiagramUploadBytes {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Image too large (max %d bytes)", maxDiagramUploadBytes))
		return
	}

	game := strings.TrimSpace(r.FormValue("game"))
	if game == "" {
		game = strings.TrimSpace(r.FormValue("type"))
	}
	if game == "" {
		game = strings.TrimSpace(r.FormValue("game_type"))
	}
	if game == "" {
		writeJSONError(w, http.StatusBadRequest, `Missing required field: "game" (chess / xianqi / shogi)`)
		return
	}
	if _, err := normalizeDiagramGameType(game, ""); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}

	filename := "diagram.png"
	if header != nil && strings.TrimSpace(header.Filename) != "" {
		filename = header.Filename
	}

	ctx, cancel := context.WithTimeout(r.Context(), diagramTimeout())
	defer cancel()

	result, err := fenFromImageByMultipart(ctx, imageBytes, filename, game)
	if err != nil {
		if proxyErr, ok := err.(*diagramProxyError); ok {
			status := proxyErr.Status
			if status < 400 || status > 599 {
				status = http.StatusBadGateway
			}
			writeJSONError(w, status, proxyErr.Message)
			return
		}
		log.Printf("api diagram fen proxy error: %v", err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Response encode error")
	}
}
