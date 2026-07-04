package store

import (
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// PubSub is the subscription handle returned by Subscribe.
// Satisfied by *redis.PubSub.
type PubSub interface {
	Channel(...redis.ChannelOption) <-chan *redis.Message
	Close() error
}

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

// NewRedisClient connects using addr. If addr is a full connection URL
// (redis://[:password@]host:port), credentials embedded in it are used;
// otherwise addr is treated as a bare host:port and REDIS_PASSWORD (if set)
// is used to authenticate.
func NewRedisClient(addr string) *redis.Client {
	if opts, err := redis.ParseURL(addr); err == nil {
		return redis.NewClient(opts)
	}
	return redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})
}
