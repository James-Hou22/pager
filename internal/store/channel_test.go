package store

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return New(rdb), mr
}

func TestCreateAndGetChannel(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	ch := Channel{
		PasswordHash:   "hash123",
		CreatedAt:      "2026-04-07T00:00:00Z",
		OrganizerToken: "tok_abc",
	}

	if err := s.CreateChannel(ctx, "ch1", ch); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}

	got, err := s.GetChannel(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetChannel: %v", err)
	}

	if got.PasswordHash != ch.PasswordHash {
		t.Errorf("PasswordHash: got %q, want %q", got.PasswordHash, ch.PasswordHash)
	}
	if got.CreatedAt != ch.CreatedAt {
		t.Errorf("CreatedAt: got %q, want %q", got.CreatedAt, ch.CreatedAt)
	}
	if got.OrganizerToken != ch.OrganizerToken {
		t.Errorf("OrganizerToken: got %q, want %q", got.OrganizerToken, ch.OrganizerToken)
	}
}

func TestGetChannel_NotFound(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	_, err := s.GetChannel(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestAddSubscriberAndGetSubscribers(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	subs := []string{
		`{"endpoint":"https://push.example.com/a","keys":{"p256dh":"k1","auth":"a1"}}`,
		`{"endpoint":"https://push.example.com/b","keys":{"p256dh":"k2","auth":"a2"}}`,
	}

	for _, sub := range subs {
		if err := s.AddSubscriber(ctx, "ch1", sub); err != nil {
			t.Fatalf("AddSubscriber: %v", err)
		}
	}

	got, err := s.GetSubscribers(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(got) != len(subs) {
		t.Errorf("subscriber count: got %d, want %d", len(got), len(subs))
	}

	seen := make(map[string]bool)
	for _, g := range got {
		seen[g] = true
	}
	for _, sub := range subs {
		if !seen[sub] {
			t.Errorf("missing subscriber: %s", sub)
		}
	}
}

func TestGetSubscribers_Empty(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	got, err := s.GetSubscribers(ctx, "ch-empty")
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %v", got)
	}
}

func TestDeleteChannel(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	ch := Channel{
		PasswordHash:   "hash123",
		CreatedAt:      "2026-04-07T00:00:00Z",
		OrganizerToken: "tok_abc",
	}
	if err := s.CreateChannel(ctx, "ch1", ch); err != nil {
		t.Fatalf("CreateChannel: %v", err)
	}
	if err := s.AddSubscriber(ctx, "ch1", `{"endpoint":"https://push.example.com/a"}`); err != nil {
		t.Fatalf("AddSubscriber: %v", err)
	}

	if err := s.DeleteChannel(ctx, "ch1"); err != nil {
		t.Fatalf("DeleteChannel: %v", err)
	}

	_, err := s.GetChannel(ctx, "ch1")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}

	subs, err := s.GetSubscribers(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetSubscribers after delete: %v", err)
	}
	if len(subs) != 0 {
		t.Errorf("expected no subscribers after delete, got %v", subs)
	}
}

func TestRemoveSubscriber(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	sub := `{"endpoint":"https://push.example.com/a","keys":{"p256dh":"k1","auth":"a1"}}`
	if err := s.AddSubscriber(ctx, "ch1", sub); err != nil {
		t.Fatalf("AddSubscriber: %v", err)
	}

	if err := s.RemoveSubscriber(ctx, "ch1", sub); err != nil {
		t.Fatalf("RemoveSubscriber: %v", err)
	}

	got, err := s.GetSubscribers(ctx, "ch1")
	if err != nil {
		t.Fatalf("GetSubscribers: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty set after remove, got %v", got)
	}
}

func TestDeleteChannel_Idempotent(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()

	// Deleting a non-existent channel should not error.
	if err := s.DeleteChannel(ctx, "ghost"); err != nil {
		t.Errorf("DeleteChannel on missing key: %v", err)
	}
}
