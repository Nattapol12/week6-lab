package handler

import (
	"errors"
	"moviehub/mw"
	"moviehub/store"
	"net/http"
)

func (h *Handler) ListWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}

	items, err := h.Store.ListWishlist(r.Context(), memberID)
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (h *Handler) AddWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}

	movieID, ok := pathID(r, "movieId")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขหนังไม่ถูกต้อง")
		return
	}

	err := h.Store.AddWishlist(r.Context(), memberID, movieID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "MOVIE_NOT_FOUND", "ไม่พบหนังเรื่องนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *Handler) RemoveWishlist(w http.ResponseWriter, r *http.Request) {
	memberID, ok := mw.MemberID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "NO_TOKEN", "กรุณาเข้าสู่ระบบก่อน")
		return
	}

	movieID, ok := pathID(r, "movieId")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขหนังไม่ถูกต้อง")
		return
	}

	err := h.Store.RemoveWishlist(r.Context(), memberID, movieID)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "WISHLIST_NOT_FOUND", "ไม่พบหนังใน Wishlist")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}
