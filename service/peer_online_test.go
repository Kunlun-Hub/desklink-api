package service

import (
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/model/custom_types"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestPeerIsOnline(t *testing.T) {
	now := time.Now().Unix()
	tests := []struct {
		name           string
		lastOnlineTime int64
		want           bool
	}{
		{name: "recent heartbeat", lastOnlineTime: now - 15, want: true},
		{name: "at timeout", lastOnlineTime: now - 60, want: true},
		{name: "missed timeout", lastOnlineTime: now - 61, want: false},
		{name: "never online", lastOnlineTime: 0, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := peerIsOnline(test.lastOnlineTime, now); got != test.want {
				t.Fatalf("peerIsOnline() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestOnlineStateIsSharedByPeerAndAddressBookLists(t *testing.T) {
	setupRecordingTest(t)
	AllService.AddressBookService = &AddressBookService{}
	if err := DB.AutoMigrate(&model.AddressBook{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	peers := []*model.Peer{
		{Id: "online-peer", Uuid: "online-uuid", LastOnlineTime: now - 15},
		{Id: "offline-peer", Uuid: "offline-uuid", LastOnlineTime: now - 61},
	}
	if err := DB.Create(&peers).Error; err != nil {
		t.Fatal(err)
	}
	addressBooks := []*model.AddressBook{
		{Id: "online-peer", UserId: 1, Tags: custom_types.AutoJson("[]")},
		{Id: "offline-peer", UserId: 1, Online: true, Tags: custom_types.AutoJson("[]")},
		{Id: "remote-peer", UserId: 1, Online: true, Tags: custom_types.AutoJson("[]")},
	}
	if err := DB.Create(&addressBooks).Error; err != nil {
		t.Fatal(err)
	}

	peerList := AllService.PeerService.List(1, 20, nil)
	peerStates := make(map[string]bool, len(peerList.Peers))
	for _, peer := range peerList.Peers {
		peerStates[peer.Id] = peer.Online
	}
	if !peerStates["online-peer"] || peerStates["offline-peer"] {
		t.Fatalf("unexpected peer states: %#v", peerStates)
	}
	onlinePeers := AllService.PeerService.List(1, 20, func(tx *gorm.DB) {
		ApplyPeerOnlineFilter(tx, "true")
	})
	if onlinePeers.Total != 1 || len(onlinePeers.Peers) != 1 || onlinePeers.Peers[0].Id != "online-peer" {
		t.Fatalf("unexpected online peer filter result: %#v", onlinePeers)
	}

	addressBookList := AllService.AddressBookService.List(1, 20, nil)
	addressBookStates := make(map[string]bool, len(addressBookList.AddressBooks))
	for _, addressBook := range addressBookList.AddressBooks {
		addressBookStates[addressBook.Id] = addressBook.Online
	}
	if !addressBookStates["online-peer"] || addressBookStates["offline-peer"] || !addressBookStates["remote-peer"] {
		t.Fatalf("unexpected address book states: %#v", addressBookStates)
	}
}

func TestMetricsHistoryIsBoundedAndChronological(t *testing.T) {
	setupRecordingTest(t)
	if err := DB.AutoMigrate(&model.AgentMetricSample{}); err != nil {
		t.Fatal(err)
	}
	if err := DB.Create([]*model.AgentMetricSample{
		{PeerId: "history-peer", Timestamp: 30, CpuUsage: 30},
		{PeerId: "history-peer", Timestamp: 10, CpuUsage: 10},
		{PeerId: "history-peer", Timestamp: 20, CpuUsage: 20},
		{PeerId: "other-peer", Timestamp: 15, CpuUsage: 99},
	}).Error; err != nil {
		t.Fatal(err)
	}
	samples, err := AllService.PeerService.MetricsHistory("history-peer", 15, 30, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != 2 || samples[0].Timestamp != 20 || samples[1].Timestamp != 30 {
		t.Fatalf("unexpected metric history: %#v", samples)
	}
}
