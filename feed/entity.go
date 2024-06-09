package feed

type FeederFetchRequest struct {
	URL string
}

type FeederFetchResponse struct {
	LatestItemTitle string
	LatestItemLink  string
}

type FeederSubscribeRequest struct {
	URL       string
	ChannelID string
}

type FeederSubscribeResponse struct {
	Title string
	Link  string
}

type FeederUnsubscribeRequest struct {
	URL       string
	ChannelID string
}

type FeederListRequest struct {
	ChannelID string
}

type FeederListResponse struct {
	Title string
	Link  string
}

type FindNewItemsResponseNewItem struct {
	Title string
	Link  string
}

type FindNewItemsResponse struct {
	Title     string
	Link      string
	ChannelID string
	NewItems  []FindNewItemsResponseNewItem
}
