package handler

import (
	"net/http"
	"strconv"
)

type PagePaginationParams struct {
	Page    int
	PerPage int
}

func ParsePagePagination(r *http.Request) PagePaginationParams {
	page := 1
	perPage := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 {
			if v <= 100 {
				perPage = v
			} else {
				perPage = 100
			}
		}
	}

	return PagePaginationParams{Page: page, PerPage: perPage}
}

func (p PagePaginationParams) Offset() int {
	return (p.Page - 1) * p.PerPage
}