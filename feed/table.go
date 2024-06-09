package feed

import "fmt"

// FeedSubscription represents a subscription to a feed.
type FeedSubscription struct {
	ID             int
	Title          string `db:"title"`
	Link           string `db:"link"`             // a rss or feed url
	LatestItemLink string `db:"latest_item_link"` // latests feed item link
	ChannelID      string `db:"channel_id"`
	CreatedAt      int64  `db:"created_at"`
	UpdatedAt      int64  `db:"updated_at"`
}

var (
	tableName = "feed_subscription"
	initTable = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	id INTEGER PRIMARY KEY,
	title TEXT,
	link TEXT,
	latest_item_link TEXT,
	channel_id TEXT,
	created_at INTEGER,
	updated_at INTEGER,
	CONSTRAINT unique_link_channel UNIQUE (link, channel_id)
);`, tableName)
)
