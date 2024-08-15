package fetcher

import (
	"context"
	"fmt"
	models "github.com/Frozelo/FeedBackManagerBot/model"
	"github.com/Frozelo/FeedBackManagerBot/rss"
	"log"
	"sync"
	"time"
)

type Sourcer interface {
	Fetch(ctx context.Context) (*[]models.Item, error)
}

type Fetcher struct {
	fetchInterval  time.Duration
	filterKeywords []string
}

func NewFetcher(interval time.Duration, filterKeywords []string) *Fetcher {
	return &Fetcher{fetchInterval: interval, filterKeywords: filterKeywords}
}

func (f *Fetcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	if err := f.Fetch(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources := []models.Source{{ID: 1, Name: "HABR", FeedURL: "https://habr.com/ru/rss/articles/"}}
	var wg sync.WaitGroup

	for _, source := range sources {
		wg.Add(1)
		go f.fetchSource(ctx, source, &wg)
	}
	wg.Wait()
	return nil
}

func (f *Fetcher) fetchSource(ctx context.Context, source models.Source, wg *sync.WaitGroup) {
	defer wg.Done()

	rssSource := rss.NewRSS(source)
	items, err := rssSource.Fetch(ctx)
	if err != nil {
		log.Printf("[ERROR] failed to fetch items from source %q: %v", source.Name, err)
		return
	}
	if err := f.processItems(ctx, items); err != nil {
		log.Printf("[ERROR] failed to process items from source %q: %v", source.Name, err)
	}
}

func (f *Fetcher) processItems(ctx context.Context, items *[]models.Item) error {
	for _, item := range *items {
		item.Date = item.Date.UTC()

		article := models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Categories:  item.Categories,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		}
		fmt.Println(article)
	}

	return nil
}
