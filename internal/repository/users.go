package repository

import (
	"context"
	models "github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UsersRepository struct {
	db *pgxpool.Pool
}

func NewUsersRepository(db *pgxpool.Pool) *UsersRepository {
	return &UsersRepository{db: db}

}

func (r *UsersRepository) GetAllUsers(ctx context.Context) ([]models.TgUser, error) {
	query := `SELECT tg_id, username FROM users`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	var users []models.TgUser
	for rows.Next() {
		var user models.TgUser
		if err = rows.Scan(&user.TgId, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (r *UsersRepository) AddTgUser(ctx context.Context, tgUser models.TgUser) error {
	query := `INSERT INTO users (tg_id, username) VALUES ($1, $2)`
	_, err := r.db.Exec(ctx, query, tgUser.TgId, tgUser.Username)
	return err
}
