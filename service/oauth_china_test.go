package service

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	appconfig "github.com/lejianwen/rustdesk-api/v2/config"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func withChinaOauthTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	oldConfig := Config
	oldDingAuth, oldDingToken, oldDingProfile := dingTalkAuthURL, dingTalkTokenURL, dingTalkProfileURL
	oldWeComAuth, oldWeComToken := weComAuthURL, weComTokenURL
	oldWeComUser, oldWeComDetail := weComUserInfoURL, weComUserDetailURL
	Config = &appconfig.Config{}
	dingTalkAuthURL = server.URL + "/dingtalk/auth"
	dingTalkTokenURL = server.URL + "/dingtalk/token"
	dingTalkProfileURL = server.URL + "/dingtalk/profile"
	weComAuthURL = server.URL + "/wecom/auth"
	weComTokenURL = server.URL + "/wecom/token"
	weComUserInfoURL = server.URL + "/wecom/userinfo"
	weComUserDetailURL = server.URL + "/wecom/detail"
	t.Cleanup(func() {
		server.Close()
		Config = oldConfig
		dingTalkAuthURL, dingTalkTokenURL, dingTalkProfileURL = oldDingAuth, oldDingToken, oldDingProfile
		weComAuthURL, weComTokenURL = oldWeComAuth, oldWeComToken
		weComUserInfoURL, weComUserDetailURL = oldWeComUser, oldWeComDetail
	})
	return server
}

func TestDingTalkAuthorizationURL(t *testing.T) {
	server := withChinaOauthTestServer(t, http.NotFoundHandler())
	info := &model.Oauth{ClientId: "ding-key", CorpId: "ding-corp"}
	authURL := dingTalkAuthorizationURL(info, "state-1", server.URL+"/callback")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("client_id") != "ding-key" || query.Get("state") != "state-1" || query.Get("scope") != "openid corpid" {
		t.Fatalf("unexpected DingTalk authorization query: %v", query)
	}
}

func TestDingTalkCallback(t *testing.T) {
	server := withChinaOauthTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/dingtalk/token":
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatal(err)
			}
			if payload["clientId"] != "ding-key" || payload["clientSecret"] != "ding-secret" || payload["code"] != "auth-code" {
				t.Fatalf("unexpected DingTalk token payload: %v", payload)
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"accessToken": "ding-token", "corpId": "ding-corp", "expireIn": 7200})
		case "/dingtalk/profile":
			if r.Header.Get("x-acs-dingtalk-access-token") != "ding-token" {
				t.Fatalf("missing DingTalk access token header")
			}
			json.NewEncoder(w).Encode(map[string]string{"nick": "DeskLink User", "openId": "ding-open", "unionId": "ding-union", "email": "ding@example.com", "avatarUrl": "https://example.com/ding.png"})
		default:
			http.NotFound(w, r)
		}
	}))
	_ = server
	service := &OauthService{}
	err, user := service.dingTalkCallback(&model.Oauth{ClientId: "ding-key", ClientSecret: "ding-secret", CorpId: "ding-corp"}, "auth-code")
	if err != nil {
		t.Fatal(err)
	}
	if user.OpenId != "ding-union" || user.Username != "ding-open" || user.Name != "DeskLink User" || user.Email != "ding@example.com" {
		t.Fatalf("unexpected DingTalk user: %#v", user)
	}
}

func TestWeComAuthorizationURL(t *testing.T) {
	server := withChinaOauthTestServer(t, http.NotFoundHandler())
	info := &model.Oauth{ClientId: "ww-corp", AgentId: "1000002"}
	authURL := weComAuthorizationURL(info, "state-2", server.URL+"/callback")
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if query.Get("appid") != "ww-corp" || query.Get("agentid") != "1000002" || query.Get("state") != "state-2" {
		t.Fatalf("unexpected WeCom authorization query: %v", query)
	}
}

func TestWeComCallback(t *testing.T) {
	server := withChinaOauthTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/wecom/token":
			if r.URL.Query().Get("corpid") != "ww-corp" || r.URL.Query().Get("corpsecret") != "ww-secret" {
				t.Fatalf("unexpected WeCom token query")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "access_token": "ww-token", "expires_in": 7200})
		case "/wecom/userinfo":
			if r.URL.Query().Get("access_token") != "ww-token" || r.URL.Query().Get("code") != "ww-code" {
				t.Fatalf("unexpected WeCom user info query")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "userid": "zhangsan"})
		case "/wecom/detail":
			if r.URL.Query().Get("userid") != "zhangsan" {
				t.Fatalf("unexpected WeCom detail query")
			}
			json.NewEncoder(w).Encode(map[string]interface{}{"errcode": 0, "userid": "zhangsan", "name": "张三", "email": "zhangsan@example.com", "avatar": "https://example.com/wecom.png"})
		default:
			http.NotFound(w, r)
		}
	}))
	_ = server
	service := &OauthService{}
	err, user := service.weComCallback(&model.Oauth{ClientId: "ww-corp", ClientSecret: "ww-secret", AgentId: "1000002"}, "ww-code")
	if err != nil {
		t.Fatal(err)
	}
	if user.OpenId != "zhangsan" || user.Username != "zhangsan" || user.Name != "张三" || user.Email != "zhangsan@example.com" {
		t.Fatalf("unexpected WeCom user: %#v", user)
	}
}

func TestChinaOauthProvidersAreDiscoverableAndKeepSecretOnUpdate(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:china-oauth-test?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Oauth{}); err != nil {
		t.Fatal(err)
	}
	oldDB := DB
	DB = database
	t.Cleanup(func() { DB = oldDB })
	service := &OauthService{}
	dingTalk := &model.Oauth{OauthType: model.OauthTypeDingTalk, ClientId: "ding-key", ClientSecret: "ding-secret"}
	weCom := &model.Oauth{OauthType: model.OauthTypeWeCom, ClientId: "ww-corp", ClientSecret: "ww-secret", AgentId: "1000002"}
	if err := service.Create(dingTalk); err != nil {
		t.Fatal(err)
	}
	if err := service.Create(weCom); err != nil {
		t.Fatal(err)
	}
	providers := service.GetOauthProviders()
	if !utils.InArray(model.OauthTypeDingTalk, providers) || !utils.InArray(model.OauthTypeWeCom, providers) {
		t.Fatalf("China OAuth providers not discoverable: %v", providers)
	}
	dingTalk.ClientSecret = ""
	dingTalk.CorpId = "corp-restricted"
	if err := service.Update(dingTalk); err != nil {
		t.Fatal(err)
	}
	updated := service.InfoById(dingTalk.Id)
	if updated.ClientSecret != "ding-secret" || updated.CorpId != "corp-restricted" {
		t.Fatalf("unexpected updated DingTalk config: %#v", updated)
	}
}
