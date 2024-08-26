package fetcher

import (
	"context"
	"github.com/Frozelo/FeedBackManagerBot/internal/model"
	"github.com/Frozelo/FeedBackManagerBot/internal/rss"
	"log"
	"sync"
	"time"
)

type SourceRepo interface {
	Sources(ctx context.Context) ([]models.Source, error)
}

type ArticleRepo interface {
	Add(ctx context.Context, article models.Article) error
}

type Sourcer interface {
	Fetch(ctx context.Context) (*[]models.Item, error)
	Id() int64
}

type Fetcher struct {
	sourceRepo     SourceRepo
	articleRepo    ArticleRepo
	fetchInterval  time.Duration
	filterKeywords []string
}

func NewFetcher(sourcesRepo SourceRepo, articleRepo ArticleRepo, interval time.Duration, filterKeywords []string) *Fetcher {
	return &Fetcher{articleRepo: articleRepo, fetchInterval: interval, filterKeywords: filterKeywords}
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

// TODO work with keywords
func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.sourceRepo.Sources(ctx)
	if err != nil {
		return err
	}
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
	if err := f.processItems(ctx, rssSource, items); err != nil {
		log.Printf("[ERROR] failed to process items from source %q: %v", source.Name, err)
	}
}

func (f *Fetcher) processItems(ctx context.Context, rssSource Sourcer, items *[]models.Item) error {
	for _, item := range *items {
		item.Date = item.Date.UTC()

		article := models.Article{
			Title: item.Title,
			// TODO implement source logic
			SourceID:    rssSource.Id(),
			Link:        item.Link,
			Categories:  item.Categories,
			PublishedAt: item.Date,
		}
		if err := f.articleRepo.Add(ctx, article); err != nil {
			return err
		}
	}

	return nil
}
