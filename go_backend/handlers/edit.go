// this comes from GO wiki page, to be extended later

package handlers

import "net/http"

func (h *Handler) Edit(w http.ResponseWriter, r *http.Request) {
	title := r.URL.Path[len("/edit/"):]
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}

	renderTemplate(w, "edit", p)
}
