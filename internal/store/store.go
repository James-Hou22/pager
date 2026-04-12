package store

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("not found")

// ErrConflict is returned when a unique constraint is violated (e.g. duplicate email).
var ErrConflict = errors.New("conflict")

// Key patterns — all Redis access must use these helpers.
func channelKey(id string) string       { return fmt.Sprintf("channel:%s", id) }
func channelSubsKey(id string) string   { return fmt.Sprintf("channel:%s:subs", id) }
func channelMsgsKey(id string) string   { return fmt.Sprintf("channel:%s:messages", id) }
func channelEventsKey(id string) string { return fmt.Sprintf("channel:%s:events", id) }

const maxMessages = 50

type Store struct {
	rdb *redis.Client
	db  *pgxpool.Pool
}

func New(rdb *redis.Client, db *pgxpool.Pool) *Store {
	return &Store{rdb: rdb, db: db}
}

func NewRedisClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}
