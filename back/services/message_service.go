package services

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"

	"warehouse-web/models"
)

// maxMessageLength 限制单条私信长度（按字符数，防止超长内容塞爆数据库）。
const maxMessageLength = 2000

// MessageService 提供私信相关的持久化操作。
type MessageService struct {
	db *gorm.DB
}

// NewMessageService 构造 MessageService。
func NewMessageService(db *gorm.DB) *MessageService {
	return &MessageService{db: db}
}

// conversationKey 把双方 ID 排序后拼成稳定的会话键 "小id:大id"。
func conversationKey(a, b uint64) string {
	if a > b {
		a, b = b, a
	}
	return strconv.FormatUint(a, 10) + ":" + strconv.FormatUint(b, 10)
}

// SendMessage 校验后持久化一条私信，返回落库后的记录。
func (s *MessageService) SendMessage(senderID, receiverID uint64, content string) (*models.Message, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "消息内容不能为空")
	}
	if utf8.RuneCountInString(content) > maxMessageLength {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "消息内容过长")
	}
	if senderID == receiverID {
		return nil, newServiceError("invalid_request", http.StatusBadRequest, "不能给自己发私信")
	}

	// 校验对方存在，避免给不存在的用户发消息
	var exists int64
	if err := s.db.Model(&models.User{}).Where("id = ?", receiverID).Count(&exists).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to validate receiver")
	}
	if exists == 0 {
		return nil, newServiceError("not_found", http.StatusNotFound, "对方用户不存在")
	}

	msg := models.Message{
		ConversationKey: conversationKey(senderID, receiverID),
		SenderID:        senderID,
		ReceiverID:      receiverID,
		Content:         content,
		CreatedAt:       time.Now().UTC(),
	}
	if err := s.db.Create(&msg).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to send message")
	}
	return &msg, nil
}

// GetThread 返回当前用户与 peer 的聊天记录（旧→新），同时把
// 对方发给我、尚未读的消息标记为已读（已读回执的核心）。
// 返回 peer 的展示信息（昵称/头像原始路径，由 controller 负责转绝对 URL）。
func (s *MessageService) GetThread(meID, peerID uint64, limit int) (models.MessagePeer, []models.Message, error) {
	if limit < 1 || limit > 200 {
		limit = 50
	}
	key := conversationKey(meID, peerID)

	// 取最近 limit 条（DESC），再反转成正序返回给前端
	var msgs []models.Message
	if err := s.db.Where("conversation_key = ?", key).
		Order("created_at DESC").
		Limit(limit).
		Find(&msgs).Error; err != nil {
		return models.MessagePeer{}, nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load messages")
	}
	for i, j := 0, len(msgs)-1; i < j; i, j = i+1, j-1 {
		msgs[i], msgs[j] = msgs[j], msgs[i]
	}

	// 标记已读：对方发给我、且未读的，统一置为现在。失败不致命。
	now := time.Now().UTC()
	_ = s.db.Model(&models.Message{}).
		Where("conversation_key = ? AND receiver_id = ? AND read_at IS NULL", key, meID).
		Update("read_at", now).Error

	peers := s.loadPeerInfos([]uint64{peerID})
	return peers[peerID], msgs, nil
}

// ListConversations 返回当前用户的所有会话（按最近一条消息时间倒序）。
// 数据量按学生项目规模，直接全量拉取后在内存聚合即可。
func (s *MessageService) ListConversations(meID uint64) ([]models.ConversationItem, error) {
	var msgs []models.Message
	if err := s.db.Where("sender_id = ? OR receiver_id = ?", meID, meID).
		Order("created_at DESC").
		Find(&msgs).Error; err != nil {
		return nil, newServiceError("internal_error", http.StatusInternalServerError, "failed to load conversations")
	}

	order := make([]uint64, 0)
	byPeer := make(map[uint64]*models.ConversationItem)

	for i := range msgs {
		m := msgs[i]
		peer := m.SenderID
		if m.SenderID == meID {
			peer = m.ReceiverID
		}
		ci, ok := byPeer[peer]
		if !ok {
			// msgs 已按时间倒序，首次出现即该会话最近一条
			ci = &models.ConversationItem{
				PeerID:      peer,
				LastMessage: m.Content,
				LastAt:      m.CreatedAt,
				LastMine:    m.SenderID == meID,
			}
			byPeer[peer] = ci
			order = append(order, peer)
		}
		if m.ReceiverID == meID && m.ReadAt == nil {
			ci.UnreadCount++
		}
	}

	if len(order) > 0 {
		peers := s.loadPeerInfos(order)
		for _, peer := range order {
			ci := byPeer[peer]
			info := peers[peer]
			ci.PeerName = info.PeerName
			ci.PeerAvatar = info.PeerAvatar
		}
	}

	result := make([]models.ConversationItem, 0, len(order))
	for _, peer := range order {
		result = append(result, *byPeer[peer])
	}
	return result, nil
}

// CountUnread 统计当前用户所有未读私信总数（导航栏小红点用）。
func (s *MessageService) CountUnread(meID uint64) (int64, error) {
	var n int64
	if err := s.db.Model(&models.Message{}).
		Where("receiver_id = ? AND read_at IS NULL", meID).
		Count(&n).Error; err != nil {
		return 0, newServiceError("internal_error", http.StatusInternalServerError, "failed to count unread")
	}
	return n, nil
}

// loadPeerInfos 批量加载一组用户的展示信息（昵称优先，回落到用户名）。
// 返回的 PeerAvatar 为数据库里的原始路径，绝对化交给 controller。
func (s *MessageService) loadPeerInfos(ids []uint64) map[uint64]models.MessagePeer {
	out := make(map[uint64]models.MessagePeer, len(ids))
	for _, id := range ids {
		out[id] = models.MessagePeer{PeerID: id}
	}

	var profiles []models.UserProfile
	if err := s.db.Where("user_id IN ?", ids).Find(&profiles).Error; err == nil {
		for _, p := range profiles {
			info := out[p.UserID]
			info.PeerName = strings.TrimSpace(p.Nickname)
			info.PeerAvatar = p.AvatarURL
			out[p.UserID] = info
		}
	}

	// 没有昵称的回落到用户名
	var users []models.User
	if err := s.db.Where("id IN ?", ids).Find(&users).Error; err == nil {
		for _, u := range users {
			info := out[u.ID]
			if info.PeerName == "" {
				info.PeerName = u.Username
			}
			out[u.ID] = info
		}
	}

	for id, info := range out {
		if info.PeerName == "" {
			info.PeerName = "同学"
			out[id] = info
		}
	}
	return out
}
