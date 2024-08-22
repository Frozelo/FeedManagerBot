package repository

import (
	"context"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SourceRepository struct {
	db *pgxpool.Pool
}

func NewSourceRepository(db *pgxpool.Pool) *SourceRepository {
	return &SourceRepository{db: db}
}

func (r *SourceRepository) Add(ctx context.Context, source models.Source) error {
	query := `INSERT INTO sources(name, feed_url, priority, created_at)
			  VALUES($1, $2, $3, $4)
			  `
	_, err := r.db.Exec(ctx, query, source.Name, source.FeedURL, source.Priority, source.CreatedAt)
	if err != nil {
		return err
	}
	return nil
}

func (r *SourceRepository) Sources(ctx context.Context) ([]models.Source, error) {
	query := `SELECT name, feed_url, priority, created_at FROM sources`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var sources []models.Source
	for rows.Next() {
		var source models.Source
		if err := rows.Scan(&source.Name, &source.FeedURL, &source.Priority, &source.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sources, nil
}
