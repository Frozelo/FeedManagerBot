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

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources := []models.Source{{ID: 1, Name: "HABR", FeedURL: "https://habr.com/ru/rss/articles/"}}
	var wg sync.WaitGroup

	for _, source := range sources {
		wg.Add(1)
		go func(source Sourcer) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("[ERROR] failed to fetch items from source %q: %v", source, err)
				return
			}
			if err := f.processItems(ctx, items); err != nil {
				log.Printf("[ERROR] failed to process items from source %q: %v", source, err)
				return
			}
		}(rss.NewRSS(source))
	}
	wg.Wait()
	return nil
}

func (f *Fetcher) processItems(ctx context.Context, items *[]models.Item) error {
	for _, item := range *items {
		item.Date = item.Date.UTC()

		fmt.Println(models.Article{
			Title:       item.Title,
			Link:        item.Link,
			Categories:  item.Categories,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		})

	}

	return nil
}
