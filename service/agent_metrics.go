package service

import (
	"errors"
	"github.com/lejianwen/rustdesk-api/v2/model"
)

const (
	defaultAgentMetricsInterval = 5
	minAgentMetricsInterval     = 5
	maxAgentMetricsInterval     = 3600
)

func AgentMetricsInterval() int {
	if DB == nil {
		return defaultAgentMetricsInterval
	}
	var setting model.AgentMetricsSetting
	if err := DB.Order("id asc").First(&setting).Error; err != nil || setting.IntervalSeconds == 0 {
		return defaultAgentMetricsInterval
	}
	return clampAgentMetricsInterval(setting.IntervalSeconds)
}

func SetAgentMetricsInterval(interval int) (int, error) {
	if interval < minAgentMetricsInterval || interval > maxAgentMetricsInterval {
		return 0, errors.New("采集频率必须在 5 到 3600 秒之间")
	}
	var setting model.AgentMetricsSetting
	result := DB.Order("id asc").First(&setting)
	if result.Error != nil {
		setting = model.AgentMetricsSetting{IntervalSeconds: interval}
		if err := DB.Create(&setting).Error; err != nil {
			return 0, err
		}
	} else if err := DB.Model(&setting).Update("interval_seconds", interval).Error; err != nil {
		return 0, err
	}
	return interval, nil
}

func clampAgentMetricsInterval(interval int) int {
	if interval < minAgentMetricsInterval {
		return minAgentMetricsInterval
	}
	if interval > maxAgentMetricsInterval {
		return maxAgentMetricsInterval
	}
	return interval
}
