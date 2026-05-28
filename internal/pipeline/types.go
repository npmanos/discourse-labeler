package types

import "encoding/json"

// RawEvent represents the top-level envelope of a Jetstream event
type RawEvent struct {
	Did    string           `json:"did"`
	TimeUS int64            `json:"time_us"`
	Type   string           `json:"type"` // "commit", "identity", "account"
	Commit *JetstreamCommit `json:"commit,omitempty"`
}

// UnmarshalJSON custom unmarshals RawEvent supporting both "type" and "kind" fields.
func (re *RawEvent) UnmarshalJSON(data []byte) error {
	type RawEventAlias struct {
		Did    string           `json:"did"`
		TimeUS int64            `json:"time_us"`
		Type   string           `json:"type"`
		Kind   string           `json:"kind"`
		Commit *JetstreamCommit `json:"commit,omitempty"`
	}
	var aux RawEventAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	re.Did = aux.Did
	re.TimeUS = aux.TimeUS
	re.Type = aux.Type
	if re.Type == "" {
		re.Type = aux.Kind
	}
	re.Commit = aux.Commit
	return nil
}

type JetstreamCommit struct {
	Rev        string          `json:"rev"`
	Type       string          `json:"type"`       // "c" (create), "u" (update), "d" (delete)
	Collection string          `json:"collection"` // e.g. "app.bsky.feed.post"
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record,omitempty"`
}

// UnmarshalJSON custom unmarshals JetstreamCommit supporting both "type" and "operation" fields.
func (jc *JetstreamCommit) UnmarshalJSON(data []byte) error {
	type JetstreamCommitAlias struct {
		Rev        string          `json:"rev"`
		Type       string          `json:"type"`
		Operation  string          `json:"operation"`
		Collection string          `json:"collection"`
		RKey       string          `json:"rkey"`
		Record     json.RawMessage `json:"record,omitempty"`
	}
	var aux JetstreamCommitAlias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	jc.Rev = aux.Rev
	jc.Type = aux.Type
	if jc.Type == "" {
		jc.Type = aux.Operation
	}
	jc.Collection = aux.Collection
	jc.RKey = aux.RKey
	jc.Record = aux.Record
	return nil
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

type ClassificationLabel string

const (
	LabelDefiniteMeta ClassificationLabel = "definite_meta"
	LabelLikelyMeta   ClassificationLabel = "likely_meta"
	LabelNotMeta      ClassificationLabel = "not_meta"
	LabelUnsure       ClassificationLabel = "unsure"
)

type PostClassification struct {
	Reasoning      string              `json:"reasoning"`
	Classification ClassificationLabel `json:"classification"`
}

type ContextAnalysis struct {
	ParentPost *PostClassification `json:"parent_post"`
	QuotePost  *PostClassification `json:"quote_post"`
}

// ClassificationResult contains LLM evaluation metrics
type ClassificationResult struct {
	Post        *HydratedPost
	Probability float64

	ContextAnalysis ContextAnalysis
	TargetPost      PostClassification
}
