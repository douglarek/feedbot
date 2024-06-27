package feed

import (
	"context"
	"fmt"
	"time"

	"github.com/gocraft/dbr/v2"
	"github.com/mmcdole/gofeed"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type Feeder struct {
	db     *dbr.Connection
	parser *gofeed.Parser
}

func New(db *dbr.Connection) *Feeder {
	sess := db.NewSession(nil)
	_, err := sess.Exec(initTable)
	if err != nil {
		panic(fmt.Errorf(fmt.Sprintf("failed to create table: %v", err)))
	}

	p := gofeed.NewParser()
	p.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36"
	return &Feeder{
		db:     db,
		parser: gofeed.NewParser(),
	}
}

func (f *Feeder) Fetch(ctx context.Context, req *FeederFetchRequest) (*FeederFetchResponse, error) {
	feed, err := f.parser.ParseURLWithContext(req.URL, ctx)
	if err != nil {
		return nil, err
	}
	return &FeederFetchResponse{
		LatestItemTitle: feed.Items[0].Title,
		LatestItemLink:  feed.Items[0].Link,
	}, nil
}

func (f *Feeder) Subscribe(ctx context.Context, req *FeederSubscribeRequest) (*FeederSubscribeResponse, error) {
	feed, err := f.parser.ParseURLWithContext(req.URL, ctx)
	if err != nil {
		return nil, err
	}

	if len(feed.Items) == 0 {
		return nil, fmt.Errorf("no items in feed")
	}

	sess := f.db.NewSession(nil)
	tx, err := sess.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUnlessCommitted()

	now := time.Now().Unix()
	if feed.Items[0].UpdatedParsed == nil {
		feed.Items[0].UpdatedParsed = feed.Items[0].PublishedParsed
	}
	_, err = tx.InsertInto(tableName).
		Columns("title", "link", "channel_id", "created_at", "updated_at").
		Values(feed.Title, req.URL, req.ChannelID, now, feed.UpdatedParsed.Unix()).ExecContext(ctx)
	if err != nil {
		if serr, ok := err.(*sqlite.Error); ok {
			if serr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return nil, fmt.Errorf("already subscribed to this feed")
			}
		}
		return nil, err
	}

	return &FeederSubscribeResponse{Title: feed.Title, Link: req.URL}, tx.Commit()
}

func (f *Feeder) Unsubscribe(ctx context.Context, req *FeederUnsubscribeRequest) error {
	tx, err := f.db.NewSession(nil).Begin()
	if err != nil {
		return err
	}
	defer tx.RollbackUnlessCommitted()

	_, err = tx.DeleteFrom(tableName).Where("channel_id = ? AND link = ?", req.ChannelID, req.URL).ExecContext(ctx)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (f *Feeder) List(ctx context.Context, req *FeederListRequest) ([]*FeederListResponse, error) {
	sess := f.db.NewSession(nil)

	var resp []*FeederListResponse
	if _, err := sess.Select("title", "link").From(tableName).
		Where("channel_id = ?", req.ChannelID).LoadContext(ctx, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

func (f *Feeder) FindNewItems(ctx context.Context) ([]*FindNewItemsResponse, error) {
	tx, err := f.db.NewSession(nil).Begin()
	if err != nil {
		return nil, err
	}
	defer tx.RollbackUnlessCommitted()

	var ft []struct {
		Link      string `db:"link"`
		ChannelID string `db:"channel_id"`
		UpdatedAt int64  `db:"updated_at"`
	}
	_, err = tx.Select("link", "channel_id", "updated_at").From(tableName).LoadContext(ctx, &ft)
	if err != nil {
		return nil, err
	}

	var resp []*FindNewItemsResponse
	var linkM = make(map[string]*gofeed.Feed) // link to feed map

	for _, row := range ft { // todo concurrency
		fd, ok := linkM[row.Link]
		if !ok {
			fd, err = f.parser.ParseURLWithContext(row.Link, ctx)
			if err != nil {
				return nil, err
			}
			linkM[row.Link] = fd
		}

		var newItems []FindNewItemsResponseNewItem
		item := fd.Items[0]
		if updated := fd.UpdatedParsed.Unix(); updated > row.UpdatedAt { // use root's updated not item's
			newItems = append(newItems, FindNewItemsResponseNewItem{
				Title: item.Title,
				Link:  item.Link,
			})
			if _, err = tx.Update(tableName).Set("updated_at", updated).
				Where("link = ? AND channel_id = ?", row.Link, row.ChannelID).ExecContext(ctx); err != nil {
				return nil, err
			}
		}

		resp = append(resp, &FindNewItemsResponse{
			Title:     fd.Title,
			Link:      row.Link,
			ChannelID: row.ChannelID,
			NewItems:  newItems,
		})
	}

	return resp, tx.Commit()
}
