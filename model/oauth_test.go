package model

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestChinaOauthTypesAreValid(t *testing.T) {
	for _, oauthType := range []string{OauthTypeDingTalk, OauthTypeWeCom} {
		if err := ValidateOauthType(oauthType); err != nil {
			t.Fatalf("%s should be a valid OAuth type: %v", oauthType, err)
		}
	}
}

func TestChinaOauthDefaults(t *testing.T) {
	dingTalk := &Oauth{OauthType: OauthTypeDingTalk}
	if err := dingTalk.FormatOauthInfo(); err != nil {
		t.Fatal(err)
	}
	if dingTalk.Op != OauthTypeDingTalk {
		t.Fatalf("DingTalk op = %q", dingTalk.Op)
	}
	weCom := &Oauth{OauthType: OauthTypeWeCom, AgentId: "1000002"}
	if err := weCom.FormatOauthInfo(); err != nil {
		t.Fatal(err)
	}
	if weCom.Op != OauthTypeWeCom {
		t.Fatalf("WeCom op = %q", weCom.Op)
	}
	if (&Oauth{OauthType: OauthTypeWeCom}).FormatOauthInfo() == nil {
		t.Fatal("WeCom without AgentID should be rejected")
	}
}

func TestOauthSecretIsNotSerialized(t *testing.T) {
	encoded, err := json.Marshal(&Oauth{ClientId: "client-id", ClientSecret: "do-not-expose"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "do-not-expose") || strings.Contains(string(encoded), "client_secret") {
		t.Fatalf("OAuth secret leaked in JSON: %s", encoded)
	}
}
