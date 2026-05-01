package handler_test

import (
	"context"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/James-Hou22/pager/internal/store"
)

func TestCloseEvent_Success(t *testing.T) {
	const eventID = "event-1"
	const orgID = "org-1"

	ms := &mockStorer{
		getEventByIDFn: func(_ context.Context, id string) (store.Event, error) {
			return store.Event{ID: id, OrganizerID: orgID}, nil
		},
		closeEventFn: func(_ context.Context, id string) (store.Event, error) {
			return store.Event{ID: id, OrganizerID: orgID, IsClosed: true}, nil
		},
	}

	app := newTestApp(ms)
	req := httptest.NewRequest("POST", "/events/"+eventID+"/close", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, orgID))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}
}

func TestCloseEvent_NotOwner(t *testing.T) {
	const eventID = "event-1"

	ms := &mockStorer{
		getEventByIDFn: func(_ context.Context, id string) (store.Event, error) {
			return store.Event{ID: id, OrganizerID: "other-org"}, nil
		},
	}

	app := newTestApp(ms)
	req := httptest.NewRequest("POST", "/events/"+eventID+"/close", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, "my-org"))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 403, got %d: %s", resp.StatusCode, body)
	}
}

func TestCloseEvent_NotFound(t *testing.T) {
	ms := &mockStorer{
		getEventByIDFn: func(_ context.Context, _ string) (store.Event, error) {
			return store.Event{}, store.ErrNotFound
		},
	}

	app := newTestApp(ms)
	req := httptest.NewRequest("POST", "/events/missing/close", nil)
	req.Header.Set("Authorization", "Bearer "+testToken(t, "org-1"))

	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != fiber.StatusNotFound {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected 404, got %d: %s", resp.StatusCode, body)
	}
}

func TestOldStatusRoutes_Removed(t *testing.T) {
	app := newTestApp(&mockStorer{})

	routes := []struct {
		method string
		path   string
	}{
		{"PATCH", "/events/evt-1/status"},
		{"PATCH", "/events/evt-1/channels/ch-1/status"},
	}

	for _, r := range routes {
		req := httptest.NewRequest(r.method, r.path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != fiber.StatusNotFound {
			t.Fatalf("%s %s: expected 404, got %d", r.method, r.path, resp.StatusCode)
		}
	}
}
