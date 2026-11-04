package handler

import (
	"encoding/json"
	"net/http"
	"testing"
)

// R-69: the public-catalog token is an admin secret; GET /setup hides it from
// other roles but they still get the fields the UI needs (name, restaurant mode).
func TestSetup_R69_CatalogTokenOnlyForAdmin(t *testing.T) {
	h := New(setupTestDB(t))
	admin := tokenFor(t, 1, "admin")
	cashier := tokenFor(t, 2, "cashier")

	w := doAuth(h, admin, http.MethodPut, "/setup", `{"name":"Tienda Uno","catalog_enabled":true,"catalog_token":"tok-abc-123","restaurant_mode":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /setup = %d, body %s", w.Code, w.Body.String())
	}

	get := func(tok string) (string, string, bool) {
		w := doAuth(h, tok, http.MethodGet, "/setup", "")
		if w.Code != http.StatusOK {
			t.Fatalf("GET /setup = %d, body %s", w.Code, w.Body.String())
		}
		var env struct {
			Data struct {
				Name           string `json:"name"`
				CatalogToken   string `json:"catalog_token"`
				RestaurantMode bool   `json:"restaurant_mode"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &env)
		return env.Data.Name, env.Data.CatalogToken, env.Data.RestaurantMode
	}

	if name, token, _ := get(admin); name != "Tienda Uno" || token != "tok-abc-123" {
		t.Errorf("admin sees name %q token %q, want Tienda Uno / tok-abc-123", name, token)
	}
	name, token, restaurant := get(cashier)
	if token != "" {
		t.Errorf("cashier sees catalog_token %q, want empty", token)
	}
	if name != "Tienda Uno" || !restaurant {
		t.Errorf("cashier lost the fields the UI needs: name %q restaurant_mode %v", name, restaurant)
	}
}
