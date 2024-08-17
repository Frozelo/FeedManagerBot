package rss

import (
	"context"
	"github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/SlyMarbo/rss"
	"log"
)

type RSS struct {
	URL      string
	SourceId int64
	Name     string
}

func NewRSS(source models.Source) *RSS {
	return &RSS{URL: source.FeedURL, SourceId: source.ID, Name: source.Name}
}

func (r *RSS) Id() int64 {
	return r.SourceId
}

func (r *RSS) Fetch(ctx context.Context) (*[]models.Item, error) {
	feed, err := loadFeed(ctx, r.URL)
	if err != nil {
		log.Printf("[ERROR] failed to load feed from %q: %v", r.URL, err)
		return nil, err
	}

	var items []models.Item
	for _, item := range feed.Items {
		itemArticle := r.createItem(item)
		items = append(items, itemArticle)
	}
	return &items, nil
}

func (r *RSS) createItem(item *rss.Item) models.Item {
	return models.Item{
		Title:      item.Title,
		Categories: item.Categories,
		Link:       item.Link,
		Date:       item.Date,
	}
}

func loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	feedCh := make(chan *rss.Feed)
	errCh := make(chan error)

	go func() {
		feed, err := rss.Fetch(url)
		if err != nil {
			errCh <- err
			return
		}
		feedCh <- feed

	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errCh:
		return nil, err
	case feed := <-feedCh:
		return feed, nil

	}

}
