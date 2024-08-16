package repository

import (
	"context"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

type ArticleRepository struct {
	db *pgxpool.Pool
}

func NewArticleRepository(db *pgxpool.Pool) *ArticleRepository {
	return &ArticleRepository{db: db}
}

func (r *ArticleRepository) Add(ctx context.Context, article models.Article) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO articles (source_id, title, link, published_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT DO NOTHING;`,
		article.SourceID,
		article.Title,
		article.Link,
		article.PublishedAt,
	)
	if err != nil {
		return err
	}

	return nil
}

func (r *ArticleRepository) GetAll(ctx context.Context) ([]models.Article, error) {
	query := `SELECT source_id, title, link, published_at FROM articles`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		if err = rows.Scan(&article.SourceID, &article.Title, &article.Link, &article.PublishedAt); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, nil
}

func (r *ArticleRepository) GetAllNotPosted(ctx context.Context) ([]models.Article, error) {
	query := `SELECT id, source_id, title, link, published_at FROM articles WHERE posted_at IS NULL`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var articles []models.Article
	for rows.Next() {
		var article models.Article
		if err = rows.Scan(&article.ID, &article.SourceID, &article.Title, &article.Link, &article.PublishedAt); err != nil {
			return nil, err
		}
		articles = append(articles, article)
	}
	return articles, nil
}

func (r *ArticleRepository) MarkAsPosted(ctx context.Context, article models.Article) error {
	log.Printf("marking article as posted")
	log.Print(article.ID)
	_, err := r.db.Exec(ctx,
		`UPDATE articles SET posted_at = $1::timestamp WHERE id = $2;`,
		time.Now().UTC(),
		article.ID,
	)
	if err != nil {
		return err
	}

	return nil
}
