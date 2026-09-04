package service

import (
	"errors"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

const (
	AccessRuleAllow = "allow"
	AccessRuleDeny  = "deny"
)

type AccessCheck struct {
	TargetPeerID string
	SourceIP     string
	SourcePeerID string
	SourceUserID uint
}

func (s *Service) CheckAccessRule(check AccessCheck) bool {
	var rules []model.AccessRule
	if DB.Where("target_peer_id = ? AND enabled = ?", check.TargetPeerID, true).Order("priority desc, id asc").Find(&rules).Error != nil {
		return true
	}
	if len(rules) == 0 {
		return true
	}
	for _, rule := range rules {
		if rule.SourceIp != "" && rule.SourceIp != check.SourceIP {
			continue
		}
		if rule.SourcePeerId != "" && rule.SourcePeerId != check.SourcePeerID {
			continue
		}
		if rule.SourceUserId != 0 && rule.SourceUserId != check.SourceUserID {
			continue
		}
		return rule.Action == AccessRuleAllow
	}
	return false
}

func (s *Service) SaveAccessRule(rule *model.AccessRule) error {
	rule.TargetPeerId = strings.TrimSpace(rule.TargetPeerId)
	rule.SourceIp = strings.TrimSpace(rule.SourceIp)
	rule.SourcePeerId = strings.TrimSpace(rule.SourcePeerId)
	rule.Action = strings.ToLower(strings.TrimSpace(rule.Action))
	if rule.TargetPeerId == "" || (rule.SourceIp == "" && rule.SourcePeerId == "" && rule.SourceUserId == 0) {
		return errors.New("target and at least one source are required")
	}
	if rule.Action != AccessRuleAllow && rule.Action != AccessRuleDeny {
		return errors.New("action must be allow or deny")
	}
	return DB.Save(rule).Error
}

func (s *Service) ListAccessRules(page, size uint) (*model.AccessRuleList, error) {
	result := &model.AccessRuleList{}
	result.Page, result.PageSize = int64(page), int64(size)
	tx := DB.Model(&model.AccessRule{})
	if err := tx.Count(&result.Total).Error; err != nil {
		return nil, err
	}
	if err := tx.Order("priority desc, id asc").Scopes(Paginate(page, size)).Find(&result.Rules).Error; err != nil {
		return nil, err
	}
	return result, nil
}
