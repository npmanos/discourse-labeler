package services

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/gorilla/websocket"
	"github.com/npmanos/discourse-labeler/internal/pipeline"
)

type ContrailsIngester struct {
	WSURL   string
	FeedURI string
}

func NewContrailsIngester(wsURL, feedURI string) *ContrailsIngester {
	return &ContrailsIngester{
		WSURL:   wsURL,
		FeedURI: feedURI,
	}
}

func (ci *ContrailsIngester) Start(ctx context.Context, cursor int64, out chan<- *types.RawEvent) error {
	u, err := url.Parse(ci.WSURL)
	if err != nil {
		return fmt.Errorf("invalid contrails WS URL: %w", err)
	}

	q := u.Query()
	q.Set("feed", ci.FeedURI)
	if cursor > 0 {
		q.Set("cursor", fmt.Sprintf("%d", cursor))
	}
	u.RawQuery = q.Encode()

	log.Printf("Connecting to Graze Contrails: %s", u.String())

	dialer := websocket.DefaultDialer
	conn, _, err := dialer.DialContext(ctx, u.String(), nil)
	if err != nil {
		return fmt.Errorf("contrails dial failure: %w", err)
	}
	defer conn.Close()

	// Start read loop
	errChan := make(chan error, 1)
	go func() {
		for {
			var ev types.RawEvent
			err := conn.ReadJSON(&ev)
			if err != nil {
				errChan <- err
				return
			}
			select {
			case out <- &ev:
			case <-ctx.Done():
				return
			}
		}
	}()

	select {
	case err := <-errChan:
		return fmt.Errorf("websocket read error: %w", err)
	case <-ctx.Done():
		return ctx.Err()
	}
}
