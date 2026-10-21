package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"moviehub/mw"
	"moviehub/store"

	"github.com/jackc/pgx/v5/pgconn"
)

func (h *Handler) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.Store.Stats(r.Context())
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, stats)
}
func (h *Handler) AdminMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Store.Pool.Query(r.Context(), `
		SELECT id, email, display_name, role, created_at
		FROM members
		ORDER BY id DESC
	`)
	if err != nil {
		internal(w, err)
		return
	}
	defer rows.Close()

	items := make([]map[string]any, 0)
	for rows.Next() {
		var id int64
		var email, name, role string
		var createdAt time.Time

		if err := rows.Scan(&id, &email, &name, &role, &createdAt); err != nil {
			internal(w, err)
			return
		}

		items = append(items, map[string]any{
			"id": id, "email": email,
			"displayName": name, "role": role,
			"createdAt": createdAt,
		})
	}
	if err := rows.Err(); err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// func (h *Handler) AdminSetRole(w http.ResponseWriter, r *http.Request) {
// 	id, ok := pathID(r, "id")
// 	if !ok {
// 		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขสมาชิกไม่ถูกต้อง")
// 		return
// 	}

// 	var req struct {
// 		Role string `json:"role"`
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		writeError(w, http.StatusBadRequest, "BAD_JSON", "ข้อมูล JSON ไม่ถูกต้อง")
// 		return
// 	}

// 	req.Role = strings.TrimSpace(req.Role)
// 	if req.Role != "member" && req.Role != "admin" {
// 		writeError(w, http.StatusBadRequest, "BAD_ROLE", "role ต้องเป็น member หรือ admin")
// 		return
// 	}

// 	tag, err := h.Store.Pool.Exec(r.Context(), `
// 		UPDATE members
// 		SET role = $1, updated_at = now()
// 		WHERE id = $2
// 	`, req.Role, id)
// 	if err != nil {
// 		internal(w, err)
// 		return
// 	}
// 	if tag.RowsAffected() == 0 {
// 		writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND", "ไม่พบสมาชิก")
// 		return
// 	}

//		writeJSON(w, http.StatusOK, map[string]any{
//			"id": id, "role": req.Role,
//		})
//	}
//
// AdminSetRole เปลี่ยนบทบาทสมาชิก โดยห้ามเปลี่ยนบทบาทตัวเอง
func (h *Handler) AdminSetRole(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID",
			"เลขสมาชิกต้องเป็นตัวเลข")
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON",
			"ข้อมูล JSON ไม่ถูกต้อง")
		return
	}

	req.Role = strings.TrimSpace(req.Role)
	if req.Role != "member" && req.Role != "admin" {
		writeError(w, http.StatusBadRequest, "BAD_ROLE",
			"role ต้องเป็น member หรือ admin")
		return
	}

	if me, _ := mw.MemberID(r.Context()); me == id {
		writeError(w, http.StatusBadRequest, "SELF_ROLE",
			"เปลี่ยนบทบาทของตัวเองไม่ได้ ให้ admin คนอื่นทำแทน")
		return
	}

	m, err := h.Store.UpdateMemberRole(r.Context(), id, req.Role)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "MEMBER_NOT_FOUND",
			"ไม่พบสมาชิกคนนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

func (h *Handler) AdminGetMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขหนังไม่ถูกต้อง")
		return
	}

	m, err := h.Store.GetMovie(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "MOVIE_NOT_FOUND", "ไม่พบหนังเรื่องนี้")
		return
	}
	if err != nil {
		internal(w, err)
		return
	}

	writeJSON(w, http.StatusOK, m)
}

type adminMovieRequest struct {
	ID               int64   `json:"id"`
	Title            string  `json:"title"`
	TitleTh          *string `json:"titleTh"`
	OriginalTitle    string  `json:"originalTitle"`
	OriginalLanguage string  `json:"originalLanguage"`
	Overview         *string `json:"overview"`
	OverviewTh       *string `json:"overviewTh"`
	PosterPath       *string `json:"posterPath"`
	BackdropPath     *string `json:"backdropPath"`
	ReleaseDate      *string `json:"releaseDate"`
	VoteAverage      float64 `json:"voteAverage"`
	VoteCount        int     `json:"voteCount"`
	Popularity       float64 `json:"popularity"`
	Adult            bool    `json:"adult"`
	Video            bool    `json:"video"`
	GenreIDs         []int   `json:"genreIds"`
}

type adminValidationError struct {
	message string
}

func (e *adminValidationError) Error() string {
	return e.message
}

func (h *Handler) AdminCreateMovie(w http.ResponseWriter, r *http.Request) {
	var req adminMovieRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "ข้อมูล JSON ไม่ถูกต้อง")
		return
	}

	if err := saveAdminMovie(r, h, 0, req); err != nil {
		writeAdminMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"id": req.ID})
}

func (h *Handler) AdminUpdateMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขหนังไม่ถูกต้อง")
		return
	}

	var req adminMovieRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_JSON", "ข้อมูล JSON ไม่ถูกต้อง")
		return
	}

	if err := saveAdminMovie(r, h, id, req); err != nil {
		writeAdminMovieError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"id": id})
}

func writeAdminMovieError(w http.ResponseWriter, err error) {
	var validation *adminValidationError
	if errors.As(err, &validation) {
		writeError(w, http.StatusBadRequest, "INVALID_MOVIE", validation.Error())
		return
	}

	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "MOVIE_NOT_FOUND", "ไม่พบหนังเรื่องนี้")
		return
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			writeError(w, http.StatusConflict, "MOVIE_EXISTS", "มีหนัง ID นี้อยู่แล้ว")
			return
		case "23503":
			writeError(w, http.StatusBadRequest, "INVALID_REFERENCE", "genreIds มีแนวหนังที่ไม่มีอยู่")
			return
		case "22007", "22008", "22009", "22015", "22P02":
			writeError(w, http.StatusBadRequest, "INVALID_MOVIE", "รูปแบบข้อมูลหนังไม่ถูกต้อง")
			return
		}
	}

	internal(w, err)
}

func (h *Handler) AdminDeleteMovie(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "BAD_ID", "เลขหนังไม่ถูกต้อง")
		return
	}

	tag, err := h.Store.Pool.Exec(r.Context(),
		`DELETE FROM movies WHERE id = $1`, id)
	if err != nil {
		internal(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		writeError(w, http.StatusNotFound, "MOVIE_NOT_FOUND", "ไม่พบหนังเรื่องนี้")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func saveAdminMovie(r *http.Request, h *Handler, pathID int64, req adminMovieRequest) error {
	req.Title = strings.TrimSpace(req.Title)
	req.OriginalTitle = strings.TrimSpace(req.OriginalTitle)
	req.OriginalLanguage = strings.TrimSpace(req.OriginalLanguage)

	id := req.ID
	if pathID > 0 {
		id = pathID
	}
	if id <= 0 || req.Title == "" || req.OriginalTitle == "" ||
		len(req.OriginalLanguage) != 2 ||
		req.VoteAverage < 0 || req.VoteAverage > 10 ||
		req.VoteCount < 0 || req.Popularity < 0 {
		return &adminValidationError{"ข้อมูลหนังไม่ถูกต้อง หรือต้องระบุ id ที่ถูกต้อง"}
	}

	if req.ReleaseDate != nil && strings.TrimSpace(*req.ReleaseDate) != "" {
		if _, err := time.Parse("2006-01-02", strings.TrimSpace(*req.ReleaseDate)); err != nil {
			return &adminValidationError{"releaseDate ต้องอยู่ในรูปแบบ YYYY-MM-DD"}
		}
	}

	seen := make(map[int]bool)
	for _, genreID := range req.GenreIDs {
		if genreID <= 0 || seen[genreID] {
			return &adminValidationError{"genreIds ต้องเป็นเลขบวกและห้ามซ้ำ"}
		}
		seen[genreID] = true
	}

	ctx := r.Context()
	tx, err := h.Store.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var releaseDate *string
	if req.ReleaseDate != nil && strings.TrimSpace(*req.ReleaseDate) != "" {
		d := strings.TrimSpace(*req.ReleaseDate)
		releaseDate = &d
	}

	if pathID == 0 {
		_, err = tx.Exec(ctx, `
			INSERT INTO movies (
				id, title, title_th, original_title, original_language,
				overview, overview_th, poster_path, backdrop_path, release_date,
				vote_average, vote_count, popularity, adult, video
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10::date,$11,$12,$13,$14,$15
			)`,
			id, req.Title, req.TitleTh, req.OriginalTitle, req.OriginalLanguage,
			req.Overview, req.OverviewTh, req.PosterPath, req.BackdropPath, releaseDate,
			req.VoteAverage, req.VoteCount, req.Popularity, req.Adult, req.Video)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `
			UPDATE movies SET
				title=$2, title_th=$3, original_title=$4, original_language=$5,
				overview=$6, overview_th=$7, poster_path=$8, backdrop_path=$9,
				release_date=$10::date, vote_average=$11, vote_count=$12,
				popularity=$13, adult=$14, video=$15, updated_at=now()
			WHERE id=$1`,
			id, req.Title, req.TitleTh, req.OriginalTitle, req.OriginalLanguage,
			req.Overview, req.OverviewTh, req.PosterPath, req.BackdropPath, releaseDate,
			req.VoteAverage, req.VoteCount, req.Popularity, req.Adult, req.Video)

		if err == nil && tag.RowsAffected() == 0 {
			return store.ErrNotFound
		}
	}
	if err != nil {
		return err
	}

	if _, err = tx.Exec(ctx, `DELETE FROM movie_genres WHERE movie_id=$1`, id); err != nil {
		return err
	}

	for position, genreID := range req.GenreIDs {
		if _, err = tx.Exec(ctx, `
			INSERT INTO movie_genres (movie_id, genre_id, position)
			VALUES ($1,$2,$3)
		`, id, genreID, position); err != nil {
			return err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	req.ID = id
	return nil
}
