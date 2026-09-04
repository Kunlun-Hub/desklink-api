package service

import (
	"github.com/lejianwen/rustdesk-api/v2/model"
	"testing"
)

func TestAgentMetricsIntervalDefaultsAndBounds(t *testing.T) {
	setupRecordingTest(t)
	if got := AgentMetricsInterval(); got != 5 {
		t.Fatalf("default interval = %d, want 5", got)
	}
	if _, err := SetAgentMetricsInterval(4); err == nil {
		t.Fatal("interval below minimum should fail")
	}
	if _, err := SetAgentMetricsInterval(3601); err == nil {
		t.Fatal("interval above maximum should fail")
	}
	if got, err := SetAgentMetricsInterval(15); err != nil || got != 15 || AgentMetricsInterval() != 15 {
		t.Fatalf("saved interval = %d, err=%v, current=%d", got, err, AgentMetricsInterval())
	}
	var setting model.AgentMetricsSetting
	if err := DB.First(&setting).Error; err != nil || setting.IntervalSeconds != 15 {
		t.Fatalf("unexpected persisted setting: %#v, %v", setting, err)
	}
}
