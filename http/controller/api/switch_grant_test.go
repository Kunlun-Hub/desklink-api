package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

func TestSwitchGrantForwardsToHbbsInternalAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/switch-grant" || r.Header.Get("X-DeskLink-Internal-Key") != "shared-key" {
			t.Fatalf("unexpected hbbs request: %s key=%q", r.URL.Path, r.Header.Get("X-DeskLink-Internal-Key"))
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["id"] != "123456789" || body["switch_code_verifier"] != "verifier" || body["signature"] != "signature" {
			t.Fatalf("unexpected forwarded body: %v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"accepted": true})
	}))
	defer internal.Close()
	oldURL := global.Config.Rustdesk.HbbsInternalUrl
	oldKey := global.Config.Rustdesk.HbbsInternalKey
	global.Config.Rustdesk.HbbsInternalUrl = internal.URL
	global.Config.Rustdesk.HbbsInternalKey = "shared-key"
	defer func() {
		global.Config.Rustdesk.HbbsInternalUrl = oldURL
		global.Config.Rustdesk.HbbsInternalKey = oldKey
	}()

	router := gin.New()
	router.POST("/api/switch-grant", (&Index{}).SwitchGrant)
	request := httptest.NewRequest(http.MethodPost, "/api/switch-grant", strings.NewReader(`{"id":"123456789","switch_code_verifier":"verifier","timestamp":"1700000000","signature":"signature"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var result map[string]interface{}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["accepted"] != true {
		t.Fatalf("unexpected response: %v", result)
	}
}
