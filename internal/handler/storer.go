package handler

import (
	"context"
	"time"

	"github.com/James-Hou22/pager/internal/store"
)

// Storer is the store interface used by Handler. Satisfied by *store.Store.
type Storer interface {
	CreateOrganizer(ctx context.Context, email, passwordHash string) (store.Organizer, error)
	GetOrganizerByEmail(ctx context.Context, email string) (store.Organizer, error)
	GetOrganizerByID(ctx context.Context, id string) (store.Organizer, error)

	GetEventByID(ctx context.Context, eventID string) (store.Event, error)
	GetEventByAccessCode(ctx context.Context, accessCode string) (store.Event, error)
	GetEventsByOrganizerID(ctx context.Context, organizerID string) ([]store.Event, error)
	CreateEvent(ctx context.Context, organizerID, name string, welcomeDescription *string, startsAt, endsAt *time.Time) (store.Event, error)
	CloseEvent(ctx context.Context, eventID string) (store.Event, error)
	GetEventByChannelID(ctx context.Context, channelID string) (store.Event, error)

	GetChannelByID(ctx context.Context, channelID string) (store.Channel, error)
	GetChannelsByEventID(ctx context.Context, eventID string) ([]store.Channel, error)
	CreateChannel(ctx context.Context, eventID, name string, description *string, opensAt, closesAt *time.Time) (store.Channel, error)

	CreateMessage(ctx context.Context, channelID, body string) (store.Message, error)
	AddMessage(ctx context.Context, channelID, message string) error
	GetMessagesByChannelID(ctx context.Context, channelID string) ([]store.Message, error)
	GetMessages(ctx context.Context, channelID string) ([]store.Message, error)

	Publish(ctx context.Context, channelID, message string) error
	Subscribe(ctx context.Context, channelID string) store.PubSub

	AddSubscriber(ctx context.Context, channelID, subJSON string) error

	GetAttendeeSessionByEndpoint(ctx context.Context, endpoint string) (string, error)
	CreateAttendeeSession(ctx context.Context, token, endpoint string) error
	GetAttendeeSessionByToken(ctx context.Context, token string) (string, error)

	AddToWaitlist(ctx context.Context, email string) error
}
