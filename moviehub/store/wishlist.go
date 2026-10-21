package store

import (
	"context"
)

// WishlistItem คือหนังในรายการที่สมาชิกอยากดู
type WishlistItem struct {
	Movie
	Note    *string `json:"note"`
	AddedAt string  `json:"addedAt"`
}

// ListWishlist อ่าน Wishlist ของสมาชิก พร้อมข้อมูลหนัง
func (s *Store) ListWishlist(ctx context.Context, memberID int64) ([]WishlistItem, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT m.id, m.title, m.title_th,
			(SELECT coalesce(g.name_th, g.name_en)
			 FROM movie_genres mg
			 JOIN genres g ON g.id = mg.genre_id
			 WHERE mg.movie_id = m.id AND mg.position = 0),
			extract(year FROM m.release_date)::int,
			round(m.vote_average, 1)::float8,
			CASE
				WHEN m.poster_path LIKE 'http%' THEN m.poster_path
				ELSE 'https://image.tmdb.org/t/p/w500' || m.poster_path
			END,
			w.note, w.added_at::text
		FROM wishlist_items w
		JOIN movies m ON m.id = w.movie_id
		WHERE w.member_id = $1
		ORDER BY w.added_at DESC`, memberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []WishlistItem{}
	for rows.Next() {
		var item WishlistItem
		if err := rows.Scan(
			&item.ID, &item.Title, &item.TitleTh,
			&item.Genre, &item.Year, &item.Rating, &item.Poster,
			&item.Note, &item.AddedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

// AddWishlist เพิ่มหนัง โดยไม่เพิ่มซ้ำหากมีอยู่แล้ว
func (s *Store) AddWishlist(ctx context.Context, memberID, movieID int64) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO wishlist_items (member_id, movie_id)
		VALUES ($1, $2)
		ON CONFLICT (member_id, movie_id) DO NOTHING`,
		memberID, movieID)
	if err != nil {
		return err
	}
	return nil
}

// RemoveWishlist ลบหนังออกจาก Wishlist
func (s *Store) RemoveWishlist(ctx context.Context, memberID, movieID int64) error {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM wishlist_items
		WHERE member_id = $1 AND movie_id = $2`,
		memberID, movieID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
