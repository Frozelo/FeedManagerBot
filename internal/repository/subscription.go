package repository

import (
	"context"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SubscriptionRepository struct {
	db *pgxpool.Pool
}

func NewSubscriberRepository(db *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Add(ctx context.Context, userId int64, sourceId int64) error {
	query := `
	INSERT INTO subscriptions (user_id, source_id) 
	VALUES ($1, $2)
	`
	_, err := r.db.Exec(ctx, query, userId, sourceId)
	if err != nil {
		return err
	}
	return nil
}

func (r *SubscriptionRepository) GetSourcesByUserID(ctx context.Context, userID int64) ([]models.Source, error) {
	query := `
        SELECT s.id, s.name, s.feed_url, s.priority, s.created_at
        FROM subscriptions sub
        JOIN sources s ON sub.source_id = s.id
        WHERE sub.user_id = $1;
    `

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sources []models.Source
	for rows.Next() {
		var source models.Source
		if err := rows.Scan(&source.ID, &source.Name, &source.FeedURL, &source.Priority, &source.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, source)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return sources, nil
}
