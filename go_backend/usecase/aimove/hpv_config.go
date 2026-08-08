// CM3070 FP code
// hpv_config.go - env config for the hpv advice client

package aimove

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

var hpvHTTPClient = &http.Client{Timeout: 0}

// hpvBaseURL - python analyser base url for history/policy/value
func hpvBaseURL() string {
	v := os.Getenv("PY_ANALYSER_URL")
	if v == "" {
		return "http://127.0.0.1:8001"
	}
	return v
}

// hpvRequestTimeout - http timeout for hpv advice calls
func hpvRequestTimeout() time.Duration {
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
