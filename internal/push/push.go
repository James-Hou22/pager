package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	webpush "github.com/SherClockHolmes/webpush-go"

	"github.com/James-Hou22/pager/internal/store"
)

const workerCount = 50

// Fanout dispatches messages to all web push subscribers for a channel.
// It uses a bounded worker pool so goroutine count is always ≤ workerCount.
type Fanout struct {
	store           *store.Store
	vapidPublicKey  string
	vapidPrivateKey string
	vapidEmail      string
}

// New reads VAPID credentials from environment variables and returns a Fanout.
// Returns an error if any required variable is missing.
func New(s *store.Store) (*Fanout, error) {
	pub := os.Getenv("VAPID_PUBLIC_KEY")
	priv := os.Getenv("VAPID_PRIVATE_KEY")
	email := os.Getenv("VAPID_EMAIL")
	if pub == "" || priv == "" || email == "" {
		return nil, fmt.Errorf("push.New: VAPID_PUBLIC_KEY, VAPID_PRIVATE_KEY, and VAPID_EMAIL must be set")
	}
	return &Fanout{
		store:           s,
		vapidPublicKey:  pub,
		vapidPrivateKey: priv,
		vapidEmail:      email,
	}, nil
}

// Send fetches all subscribers for channelID and delivers message to each
// using a worker pool. Per-subscriber errors are logged, not returned.
func (f *Fanout) Send(ctx context.Context, channelID, message string) {
	subs, err := f.store.GetSubscribers(ctx, channelID)
	if err != nil {
		log.Printf("push.Send: get subscribers for %s: %v", channelID, err)
		return
	}
	if len(subs) == 0 {
		return
	}

	jobs := make(chan string, len(subs))
	for _, sub := range subs {
		jobs <- sub
	}
	close(jobs)

	n := min(workerCount, len(subs))
	var wg sync.WaitGroup
	wg.Add(n)
	for range n {
		go func() {
			defer wg.Done()
			for subJSON := range jobs {
				f.sendOne(ctx, channelID, subJSON, message)
			}
		}()
	}
	wg.Wait()
}

// sendOne delivers a single web push notification.
// On 410 Gone the subscription is pruned from Redis; all other non-2xx errors are logged.
func (f *Fanout) sendOne(ctx context.Context, channelID, subJSON, message string) {
	var sub webpush.Subscription
	if err := json.Unmarshal([]byte(subJSON), &sub); err != nil {
		log.Printf("push.sendOne: unmarshal subscription: %v", err)
		return
	}

	resp, err := webpush.SendNotificationWithContext(ctx, []byte(message), &sub, &webpush.Options{
		VAPIDPublicKey:  f.vapidPublicKey,
		VAPIDPrivateKey: f.vapidPrivateKey,
		Subscriber:      f.vapidEmail,
		TTL:             30,
	})
	if err != nil {
		log.Printf("push.sendOne: send to %s: %v", sub.Endpoint, err)
		return
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusGone:
		// Subscriber has unsubscribed; remove from Redis so we stop sending to them.
		if err := f.store.RemoveSubscriber(ctx, channelID, subJSON); err != nil {
			log.Printf("push.sendOne: remove stale subscriber %s: %v", sub.Endpoint, err)
		} else {
			log.Printf("push.sendOne: removed stale subscriber %s", sub.Endpoint)
		}
	case http.StatusCreated, http.StatusOK:
		// Success — push services typically return 201.
	default:
		log.Printf("push.sendOne: push service returned %d for %s", resp.StatusCode, sub.Endpoint)
	}
}
