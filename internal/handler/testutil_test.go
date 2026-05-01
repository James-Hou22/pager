package handler_test

import (
	"context"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gofiber/fiber/v2"
	"testing"

	"github.com/James-Hou22/pager/internal/handler"
	"github.com/James-Hou22/pager/internal/store"
)

var testJWTSecret = []byte("test-secret")

func testToken(t *testing.T, organizerID string) string {
	t.Helper()
	claims := jwt.MapClaims{
		"organizer_id": organizerID,
		"exp":          time.Now().Add(time.Hour).Unix(),
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(testJWTSecret)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}

func newTestApp(s handler.Storer) *fiber.App {
	h := handler.New(s, nil, testJWTSecret)
	app := fiber.New()
	h.Register(app)
	return app
}

// mockStorer implements handler.Storer with optional function fields.
// Any unset field panics with the method name if called.
type mockStorer struct {
	createOrganizerFn              func(context.Context, string, string) (store.Organizer, error)
	getOrganizerByEmailFn          func(context.Context, string) (store.Organizer, error)
	getOrganizerByIDFn             func(context.Context, string) (store.Organizer, error)
	getEventByIDFn                 func(context.Context, string) (store.Event, error)
	getEventByAccessCodeFn         func(context.Context, string) (store.Event, error)
	getEventsByOrganizerIDFn       func(context.Context, string) ([]store.Event, error)
	createEventFn                  func(context.Context, string, string, *string, *time.Time, *time.Time) (store.Event, error)
	closeEventFn                   func(context.Context, string) (store.Event, error)
	getEventByChannelIDFn          func(context.Context, string) (store.Event, error)
	getChannelByIDFn               func(context.Context, string) (store.Channel, error)
	getChannelsByEventIDFn         func(context.Context, string) ([]store.Channel, error)
	createChannelFn                func(context.Context, string, string, *string, *time.Time, *time.Time) (store.Channel, error)
	createMessageFn                func(context.Context, string, string) (store.Message, error)
	addMessageFn                   func(context.Context, string, string) error
	getMessagesByChannelIDFn       func(context.Context, string) ([]store.Message, error)
	getMessagesFn                  func(context.Context, string) ([]store.Message, error)
	publishFn                      func(context.Context, string, string) error
	subscribeFn                    func(context.Context, string) store.PubSub
	addSubscriberFn                func(context.Context, string, string) error
	getAttendeeSessionByEndpointFn func(context.Context, string) (string, error)
	createAttendeeSessionFn        func(context.Context, string, string) error
	getAttendeeSessionByTokenFn    func(context.Context, string) (string, error)
	addToWaitlistFn                func(context.Context, string) error
}

func (m *mockStorer) CreateOrganizer(ctx context.Context, email, hash string) (store.Organizer, error) {
	if m.createOrganizerFn != nil {
		return m.createOrganizerFn(ctx, email, hash)
	}
	panic("CreateOrganizer not set on mock")
}
func (m *mockStorer) GetOrganizerByEmail(ctx context.Context, email string) (store.Organizer, error) {
	if m.getOrganizerByEmailFn != nil {
		return m.getOrganizerByEmailFn(ctx, email)
	}
	panic("GetOrganizerByEmail not set on mock")
}
func (m *mockStorer) GetOrganizerByID(ctx context.Context, id string) (store.Organizer, error) {
	if m.getOrganizerByIDFn != nil {
		return m.getOrganizerByIDFn(ctx, id)
	}
	panic("GetOrganizerByID not set on mock")
}
func (m *mockStorer) GetEventByID(ctx context.Context, id string) (store.Event, error) {
	if m.getEventByIDFn != nil {
		return m.getEventByIDFn(ctx, id)
	}
	panic("GetEventByID not set on mock")
}
func (m *mockStorer) GetEventByAccessCode(ctx context.Context, code string) (store.Event, error) {
	if m.getEventByAccessCodeFn != nil {
		return m.getEventByAccessCodeFn(ctx, code)
	}
	panic("GetEventByAccessCode not set on mock")
}
func (m *mockStorer) GetEventsByOrganizerID(ctx context.Context, id string) ([]store.Event, error) {
	if m.getEventsByOrganizerIDFn != nil {
		return m.getEventsByOrganizerIDFn(ctx, id)
	}
	panic("GetEventsByOrganizerID not set on mock")
}
func (m *mockStorer) CreateEvent(ctx context.Context, orgID, name string, desc *string, sa, ea *time.Time) (store.Event, error) {
	if m.createEventFn != nil {
		return m.createEventFn(ctx, orgID, name, desc, sa, ea)
	}
	panic("CreateEvent not set on mock")
}
func (m *mockStorer) CloseEvent(ctx context.Context, id string) (store.Event, error) {
	if m.closeEventFn != nil {
		return m.closeEventFn(ctx, id)
	}
	panic("CloseEvent not set on mock")
}
func (m *mockStorer) GetEventByChannelID(ctx context.Context, id string) (store.Event, error) {
	if m.getEventByChannelIDFn != nil {
		return m.getEventByChannelIDFn(ctx, id)
	}
	panic("GetEventByChannelID not set on mock")
}
func (m *mockStorer) GetChannelByID(ctx context.Context, id string) (store.Channel, error) {
	if m.getChannelByIDFn != nil {
		return m.getChannelByIDFn(ctx, id)
	}
	panic("GetChannelByID not set on mock")
}
func (m *mockStorer) GetChannelsByEventID(ctx context.Context, id string) ([]store.Channel, error) {
	if m.getChannelsByEventIDFn != nil {
		return m.getChannelsByEventIDFn(ctx, id)
	}
	panic("GetChannelsByEventID not set on mock")
}
func (m *mockStorer) CreateChannel(ctx context.Context, eventID, name string, desc *string, oa, ca *time.Time) (store.Channel, error) {
	if m.createChannelFn != nil {
		return m.createChannelFn(ctx, eventID, name, desc, oa, ca)
	}
	panic("CreateChannel not set on mock")
}
func (m *mockStorer) CreateMessage(ctx context.Context, channelID, body string) (store.Message, error) {
	if m.createMessageFn != nil {
		return m.createMessageFn(ctx, channelID, body)
	}
	panic("CreateMessage not set on mock")
}
func (m *mockStorer) AddMessage(ctx context.Context, channelID, msg string) error {
	if m.addMessageFn != nil {
		return m.addMessageFn(ctx, channelID, msg)
	}
	panic("AddMessage not set on mock")
}
func (m *mockStorer) GetMessagesByChannelID(ctx context.Context, id string) ([]store.Message, error) {
	if m.getMessagesByChannelIDFn != nil {
		return m.getMessagesByChannelIDFn(ctx, id)
	}
	panic("GetMessagesByChannelID not set on mock")
}
func (m *mockStorer) GetMessages(ctx context.Context, id string) ([]store.Message, error) {
	if m.getMessagesFn != nil {
		return m.getMessagesFn(ctx, id)
	}
	panic("GetMessages not set on mock")
}
func (m *mockStorer) Publish(ctx context.Context, channelID, msg string) error {
	if m.publishFn != nil {
		return m.publishFn(ctx, channelID, msg)
	}
	panic("Publish not set on mock")
}
func (m *mockStorer) Subscribe(ctx context.Context, channelID string) store.PubSub {
	if m.subscribeFn != nil {
		return m.subscribeFn(ctx, channelID)
	}
	return nil
}
func (m *mockStorer) AddSubscriber(ctx context.Context, channelID, sub string) error {
	if m.addSubscriberFn != nil {
		return m.addSubscriberFn(ctx, channelID, sub)
	}
	panic("AddSubscriber not set on mock")
}
func (m *mockStorer) GetAttendeeSessionByEndpoint(ctx context.Context, ep string) (string, error) {
	if m.getAttendeeSessionByEndpointFn != nil {
		return m.getAttendeeSessionByEndpointFn(ctx, ep)
	}
	panic("GetAttendeeSessionByEndpoint not set on mock")
}
func (m *mockStorer) CreateAttendeeSession(ctx context.Context, token, ep string) error {
	if m.createAttendeeSessionFn != nil {
		return m.createAttendeeSessionFn(ctx, token, ep)
	}
	panic("CreateAttendeeSession not set on mock")
}
func (m *mockStorer) GetAttendeeSessionByToken(ctx context.Context, token string) (string, error) {
	if m.getAttendeeSessionByTokenFn != nil {
		return m.getAttendeeSessionByTokenFn(ctx, token)
	}
	panic("GetAttendeeSessionByToken not set on mock")
}
func (m *mockStorer) AddToWaitlist(ctx context.Context, email string) error {
	if m.addToWaitlistFn != nil {
		return m.addToWaitlistFn(ctx, email)
	}
	panic("AddToWaitlist not set on mock")
}
