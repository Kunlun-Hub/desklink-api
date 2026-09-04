package service

import (
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAccessRulesDefaultAllowAndExplicitAllowlist(t *testing.T) {
	setupRecordingTest(t)
	if !AllService.CheckAccessRule(AccessCheck{TargetPeerID: "unmanaged"}) {
		t.Fatal("targets without rules must preserve existing access behavior")
	}
	rules := []*model.AccessRule{
		{TargetPeerId: "target", SourceIp: "10.202.119.252", Action: AccessRuleAllow, Priority: 100, Enabled: true},
		{TargetPeerId: "target", SourceUserId: 1, Action: AccessRuleAllow, Priority: 90, Enabled: true},
	}
	for _, rule := range rules {
		if err := AllService.SaveAccessRule(rule); err != nil {
			t.Fatal(err)
		}
	}
	if !AllService.CheckAccessRule(AccessCheck{TargetPeerID: "target", SourceIP: "10.202.119.252"}) {
		t.Fatal("allowed source IP was denied")
	}
	if !AllService.CheckAccessRule(AccessCheck{TargetPeerID: "target", SourceUserID: 1}) {
		t.Fatal("allowed user was denied")
	}
	if AllService.CheckAccessRule(AccessCheck{TargetPeerID: "target", SourceIP: "10.202.119.253"}) {
		t.Fatal("unmatched source was not denied")
	}
}

func TestAccessRulePriorityAndCombinedConditions(t *testing.T) {
	setupRecordingTest(t)
	deny := &model.AccessRule{TargetPeerId: "target", SourceIp: "10.0.0.1", Action: AccessRuleDeny, Priority: 200, Enabled: true}
	allow := &model.AccessRule{TargetPeerId: "target", SourceIp: "10.0.0.1", SourceUserId: 7, Action: AccessRuleAllow, Priority: 100, Enabled: true}
	if err := AllService.SaveAccessRule(deny); err != nil {
		t.Fatal(err)
	}
	if err := AllService.SaveAccessRule(allow); err != nil {
		t.Fatal(err)
	}
	if AllService.CheckAccessRule(AccessCheck{TargetPeerID: "target", SourceIP: "10.0.0.1", SourceUserID: 7}) {
		t.Fatal("higher-priority deny was ignored")
	}
	invalid := &model.AccessRule{TargetPeerId: "target", Action: AccessRuleAllow}
	if AllService.SaveAccessRule(invalid) == nil {
		t.Fatal("source-less rule was accepted")
	}
}
