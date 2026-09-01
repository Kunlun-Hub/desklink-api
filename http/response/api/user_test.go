package api

import (
	"encoding/json"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func boolPointer(value bool) *bool {
	return &value
}

func TestUserPayloadMatchesDeskLink149Contract(t *testing.T) {
	user := &model.User{
		Username: "desklink-user",
		Nickname: "DeskLink User",
		Avatar:   "https://assets.example/avatar.png",
		Email:    "user@example.com",
		IsAdmin:  boolPointer(true),
		Status:   model.COMMON_STATUS_ENABLE,
	}

	payload := (&UserPayload{}).FromUser(user)
	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(encoded, &response); err != nil {
		t.Fatal(err)
	}

	checks := map[string]interface{}{
		"name":         "desklink-user",
		"display_name": "DeskLink User",
		"avatar":       "https://assets.example/avatar.png",
		"status":       float64(1),
		"is_admin":     true,
	}
	for key, expected := range checks {
		if actual := response[key]; actual != expected {
			t.Errorf("%s = %#v, want %#v", key, actual, expected)
		}
	}
}

func TestDisabledUserUsesRustDeskStatusZero(t *testing.T) {
	user := &model.User{Status: model.COMMON_STATUS_DISABLED}
	payload := (&UserPayload{}).FromUser(user)
	if payload.Status != 0 {
		t.Fatalf("status = %d, want 0 for a disabled 1.4.9 user", payload.Status)
	}
}
