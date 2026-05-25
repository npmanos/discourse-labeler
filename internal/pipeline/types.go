package types

import "encoding/json"

// RawEvent represents the top-level envelope of a Jetstream event
type RawEvent struct {
	Did    string           `json:"did"`
	TimeUS int64            `json:"time_us"`
	Type   string           `json:"type"` // "commit", "identity", "account"
	Commit *JetstreamCommit `json:"commit,omitempty"`
}

type JetstreamCommit struct {
	Rev        string          `json:"rev"`
	Type       string          `json:"type"`       // "c" (create), "u" (update), "d" (delete)
	Collection string          `json:"collection"` // e.g. "app.bsky.feed.post"
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record,omitempty"`
}

type BskyPostRecord struct {
	Type      string     `json:"$type"`
	CreatedAt string     `json:"createdAt"`
	Text      string     `json:"text"`
	Reply     *BskyReply `json:"reply,omitempty"`
	Embed     *BskyEmbed `json:"embed,omitempty"`
}

type BskyReply struct {
	Parent *BskyLink `json:"parent,omitempty"`
	Root   *BskyLink `json:"root,omitempty"`
}

type BskyEmbed struct {
	Type   string    `json:"$type"` // e.g. "app.bsky.embed.record"
	Record *BskyLink `json:"record,omitempty"`
}

type BskyLink struct {
	CID string `json:"cid"`
	URI string `json:"uri"`
}

// HydratedPost carries resolved context alongside the target post
type HydratedPost struct {
	TargetDID        string
	TargetRKey       string
	TargetURI        string
	TargetText       string
	ParentText       string
	QuotedText       string
	HasParentContext bool
	EventTimeUS      int64
}

// ClassificationResult contains LLM evaluation metrics
type ClassificationResult struct {
	Post            *HydratedPost
	IsMetaDiscourse bool
	Probability     float64
}
