package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func sharedRoleJSON(id, name, role string) map[string]any {
	return map[string]any{
		"id":                 id,
		"name":               name,
		"shareRecipientType": "group",
		"role":               role,
	}
}

// A deployment can carry more grants than fit on one page. Reading only the
// first page made a grant on a later page look revoked, so the resource dropped
// it from state and the next apply tried to recreate a grant that still existed.
func TestListDeploymentSharedRolesReadsEveryPage(t *testing.T) {
	var paths []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.RequestURI())

		w.Header().Set("Content-Type", "application/json")
		body := map[string]any{}

		switch r.URL.Query().Get("offset") {
		case "":
			body["data"] = []map[string]any{sharedRoleJSON("grp-1", "FirstPage", "CONSUMER")}
			// Only the query of next is used, the rest is rebuilt from the path.
			body["next"] = "https://app.datarobot.com/api/v2/deployments/dep-1/sharedRoles/?offset=1&limit=1"
		case "1":
			body["data"] = []map[string]any{sharedRoleJSON("grp-2", "SecondPage", "USER")}
			body["next"] = ""
		default:
			t.Errorf("unexpected offset %q", r.URL.Query().Get("offset"))
		}

		_ = json.NewEncoder(w).Encode(body)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	roles, err := svc.ListDeploymentSharedRoles(context.Background(), "dep-1")
	if err != nil {
		t.Fatalf("ListDeploymentSharedRoles returned error: %v", err)
	}

	if len(roles) != 2 {
		t.Fatalf("expected both pages to be read, got %d grant(s): %+v", len(roles), roles)
	}
	if roles[0].ID != "grp-1" || roles[1].ID != "grp-2" {
		t.Errorf("expected grp-1 then grp-2, got %q then %q", roles[0].ID, roles[1].ID)
	}
	if roles[1].Role != "USER" {
		t.Errorf("expected the second page's role to survive, got %q", roles[1].Role)
	}
	if len(paths) != 2 {
		t.Errorf("expected two requests, got %d: %v", len(paths), paths)
	}
}

func TestListDeploymentSharedRolesEscapesTheDeploymentID(t *testing.T) {
	var path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.EscapedPath()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}, "next": ""})
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	if _, err := svc.ListDeploymentSharedRoles(context.Background(), "dep/1"); err != nil {
		t.Fatalf("ListDeploymentSharedRoles returned error: %v", err)
	}

	if want := "/deployments/dep%2F1/sharedRoles/"; path != want {
		t.Errorf("expected the ID to be escaped into one path segment (%s), got %s", want, path)
	}
}

func TestUpdateDeploymentSharedRolesSendsAnUpdateRolesBody(t *testing.T) {
	var method string
	var body UpdateSharedRolesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("could not decode the request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := NewConfiguration("fake-token")
	cfg.Endpoint = server.URL
	svc := NewService(NewClient(cfg))

	err := svc.UpdateDeploymentSharedRoles(context.Background(), "dep-1", &UpdateSharedRolesRequest{
		Operation: SharedRolesOperationUpdate,
		Roles: []SharedRoleGrant{
			{ShareRecipientType: ShareRecipientTypeGroup, ID: "grp-1", Role: SharedRoleNone},
		},
	})
	if err != nil {
		t.Fatalf("UpdateDeploymentSharedRoles returned error: %v", err)
	}

	if method != http.MethodPatch {
		t.Errorf("expected PATCH, got %s", method)
	}
	if body.Operation != "updateRoles" {
		t.Errorf("expected the updateRoles operation, got %q", body.Operation)
	}
	if len(body.Roles) != 1 {
		t.Fatalf("expected one grant in the body, got %d", len(body.Roles))
	}
	if body.Roles[0].Role != "NO_ROLE" {
		t.Errorf("expected NO_ROLE to be the revoke value, got %q", body.Roles[0].Role)
	}
	if body.Roles[0].ShareRecipientType != "group" {
		t.Errorf("expected a group recipient, got %q", body.Roles[0].ShareRecipientType)
	}
}
