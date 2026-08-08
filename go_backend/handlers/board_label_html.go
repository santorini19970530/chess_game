// CM3070 FP code
// board_label_html.go - file and rank label html for board ssr

package handlers

import (
	"fmt"
	"strings"
)

// generateFileLabels - builds file letter labels (a…)
func generateFileLabels(files int) string {
	var b strings.Builder
	for i := 0; i < files; i++ {
		b.WriteString(`<span class="board_label">`)
		b.WriteByte(byte('a' + i))
		b.WriteString(`</span>`)
	}
	return b.String()
}

// generateRankLabels - builds rank number labels (n…1)
func generateRankLabels(ranks int) string {
	var b strings.Builder
	for r := ranks; r >= 1; r-- {
		b.WriteString(`<span class="board_label">`)
		b.WriteString(fmt.Sprintf("%d", r))
		b.WriteString(`</span>`)
	}
	return b.String()
}
