package controllers

import (
	"net/http"
	"ranking-school/utils"
)

type Controller struct {
}

func (c Controller) ProtectedEndpoint() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.ResponseJSON(w, "YES")
	}
}
