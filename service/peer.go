package service

import (
	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
	"time"
)

const peerOnlineTimeout = 60 * time.Second

type PeerService struct {
}

func peerIsOnline(lastOnlineTime, now int64) bool {
	return lastOnlineTime > 0 && now-lastOnlineTime <= int64(peerOnlineTimeout/time.Second)
}

func (ps *PeerService) IsOnline(lastOnlineTime int64) bool {
	return peerIsOnline(lastOnlineTime, time.Now().Unix())
}

func (ps *PeerService) MetricsHistory(peerID string, from, to int64, limit int) ([]*model.AgentMetricSample, error) {
	if limit <= 0 || limit > 2000 {
		limit = 1000
	}
	tx := DB.Where("peer_id = ?", peerID)
	if from > 0 {
		tx = tx.Where("timestamp >= ?", from)
	}
	if to > 0 {
		tx = tx.Where("timestamp <= ?", to)
	}
	var count int64
	if err := tx.Model(&model.AgentMetricSample{}).Count(&count).Error; err != nil {
		return nil, err
	}
	if count <= int64(limit) {
		var samples []*model.AgentMetricSample
		return samples, tx.Order("timestamp asc, id asc").Find(&samples).Error
	}
	var ids []uint
	if err := tx.Model(&model.AgentMetricSample{}).Order("timestamp asc, id asc").Pluck("id", &ids).Error; err != nil {
		return nil, err
	}
	selectedIDs := make([]uint, 0, limit)
	if limit == 1 {
		selectedIDs = append(selectedIDs, ids[len(ids)-1])
	} else {
		for index := 0; index < limit; index++ {
			position := index * (len(ids) - 1) / (limit - 1)
			selectedIDs = append(selectedIDs, ids[position])
		}
	}
	var samples []*model.AgentMetricSample
	err := DB.Where("id IN ?", selectedIDs).Order("timestamp asc, id asc").Find(&samples).Error
	return samples, err
}

func ApplyPeerOnlineFilter(tx *gorm.DB, online string) {
	cutoff := time.Now().Unix() - int64(peerOnlineTimeout/time.Second)
	switch online {
	case "true":
		tx.Where("last_online_time >= ?", cutoff)
	case "false":
		tx.Where("last_online_time < ?", cutoff)
	}
}

func (ps *PeerService) OnlineStatesByIds(ids []string) map[string]bool {
	states := make(map[string]bool, len(ids))
	if len(ids) == 0 {
		return states
	}
	var peers []*model.Peer
	DB.Select("id", "last_online_time").Where("id in ?", ids).Find(&peers)
	now := time.Now().Unix()
	for _, peer := range peers {
		states[peer.Id] = peerIsOnline(peer.LastOnlineTime, now)
	}
	return states
}

// FindById 根据id查找
func (ps *PeerService) FindById(id string) *model.Peer {
	p := &model.Peer{}
	DB.Where("id = ?", id).First(p)
	return p
}
func (ps *PeerService) FindByUuid(uuid string) *model.Peer {
	p := &model.Peer{}
	DB.Where("uuid = ?", uuid).First(p)
	return p
}
func (ps *PeerService) InfoByRowId(id uint) *model.Peer {
	p := &model.Peer{}
	DB.Where("row_id = ?", id).First(p)
	return p
}

// FindByUserIdAndUuid 根据用户id和uuid查找peer
func (ps *PeerService) FindByUserIdAndUuid(uuid string, userId uint) *model.Peer {
	p := &model.Peer{}
	DB.Where("uuid = ? and user_id = ?", uuid, userId).First(p)
	return p
}

// UuidBindUserId 绑定用户id
func (ps *PeerService) UuidBindUserId(deviceId string, uuid string, userId uint) {
	peer := ps.FindByUuid(uuid)
	// 如果存在则更新
	if peer.RowId > 0 {
		peer.UserId = userId
		ps.Update(peer)
	} else {
		// 不存在则创建
		/*if deviceId != "" {
			DB.Create(&model.Peer{
				Id:     deviceId,
				Uuid:   uuid,
				UserId: userId,
			})
		}*/
	}
}

// UuidUnbindUserId 解绑用户id, 用于用户注销
func (ps *PeerService) UuidUnbindUserId(uuid string, userId uint) {
	peer := ps.FindByUserIdAndUuid(uuid, userId)
	if peer.RowId > 0 {
		DB.Model(peer).Update("user_id", 0)
	}
}

// EraseUserId 清除用户id, 用于用户删除
func (ps *PeerService) EraseUserId(userId uint) error {
	return DB.Model(&model.Peer{}).Where("user_id = ?", userId).Update("user_id", 0).Error
}

// ListByUserIds 根据用户id取列表
func (ps *PeerService) ListByUserIds(userIds []uint, page, pageSize uint) (res *model.PeerList) {
	res = &model.PeerList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.Peer{})
	tx.Where("user_id in (?)", userIds)
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.Peers)
	now := time.Now().Unix()
	for _, peer := range res.Peers {
		peer.Online = peerIsOnline(peer.LastOnlineTime, now)
	}
	return
}

func (ps *PeerService) List(page, pageSize uint, where func(tx *gorm.DB)) (res *model.PeerList) {
	res = &model.PeerList{}
	res.Page = int64(page)
	res.PageSize = int64(pageSize)
	tx := DB.Model(&model.Peer{})
	if where != nil {
		where(tx)
	}
	tx.Count(&res.Total)
	tx.Scopes(Paginate(page, pageSize))
	tx.Find(&res.Peers)
	now := time.Now().Unix()
	for _, peer := range res.Peers {
		peer.Online = peerIsOnline(peer.LastOnlineTime, now)
	}
	return
}

// ListFilterByUserId 根据用户id过滤Peer列表
func (ps *PeerService) ListFilterByUserId(page, pageSize uint, where func(tx *gorm.DB), userId uint) (res *model.PeerList) {
	userWhere := func(tx *gorm.DB) {
		tx.Where("user_id = ?", userId)
		// 如果还有额外的筛选条件，执行它
		if where != nil {
			where(tx)
		}
	}
	return ps.List(page, pageSize, userWhere)
}

// Create 创建
func (ps *PeerService) Create(u *model.Peer) error {
	res := DB.Create(u).Error
	return res
}

// Delete 删除, 同时也应该删除token
func (ps *PeerService) Delete(u *model.Peer) error {
	uuid := u.Uuid
	err := DB.Delete(u).Error
	if err != nil {
		return err
	}
	if err = DB.Where("peer_id = ?", u.Id).Delete(&model.DeviceCredential{}).Error; err != nil {
		return err
	}
	// 删除token
	return AllService.UserService.FlushTokenByUuid(uuid)
}

// GetUuidListByIDs 根据ids获取uuid列表
func (ps *PeerService) GetUuidListByIDs(ids []uint) ([]string, error) {
	var uuids []string
	err := DB.Model(&model.Peer{}).
		Where("row_id in (?)", ids).
		Pluck("uuid", &uuids).Error
	//过滤uuids中的空字符串
	var newUuids []string
	for _, uuid := range uuids {
		if uuid != "" {
			newUuids = append(newUuids, uuid)
		}
	}
	return newUuids, err
}

// BatchDelete 批量删除, 同时也应该删除token
func (ps *PeerService) BatchDelete(ids []uint) error {
	uuids, err := ps.GetUuidListByIDs(ids)
	var peerIds []string
	if err = DB.Model(&model.Peer{}).Where("row_id in (?)", ids).Pluck("id", &peerIds).Error; err != nil {
		return err
	}
	err = DB.Where("row_id in (?)", ids).Delete(&model.Peer{}).Error
	if err != nil {
		return err
	}
	if err = DB.Where("peer_id in (?)", peerIds).Delete(&model.DeviceCredential{}).Error; err != nil {
		return err
	}
	// 删除token
	return AllService.UserService.FlushTokenByUuids(uuids)
}

// Update 更新
func (ps *PeerService) Update(u *model.Peer) error {
	return DB.Model(u).Updates(u).Error
}
