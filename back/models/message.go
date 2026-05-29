package models

import "time"

// Message 表示用户之间的一条私信。
//
// ConversationKey 是把双方 ID 排序后拼成的 "小id:大id"，
// 让"取某段会话""数未读"都能用单一索引列查询，不必 OR 两个方向。
type Message struct {
	ID              uint64     `gorm:"primaryKey;autoIncrement"      json:"id"`
	ConversationKey string     `gorm:"size:64;index"                 json:"-"`
	SenderID        uint64     `gorm:"index"                         json:"sender_id"`
	ReceiverID      uint64     `gorm:"index"                         json:"receiver_id"`
	Content         string     `gorm:"type:text"                     json:"content"`
	ReadAt          *time.Time `json:"read_at,omitempty"` // nil = 未读（接收方尚未打开会话）
	CreatedAt       time.Time  `json:"created_at"`
}

// MessageItem 是聊天记录里单条消息的响应视图。
//   - Mine：是否当前用户发出（前端据此左右分栏）
//   - Read：是否已读（对"我发出的"消息才有"已读/未读"语义）
type MessageItem struct {
	ID         uint64    `json:"id"`
	SenderID   uint64    `json:"sender_id"`
	ReceiverID uint64    `json:"receiver_id"`
	Content    string    `json:"content"`
	Mine       bool      `json:"mine"`
	Read       bool      `json:"read"`
	CreatedAt  time.Time `json:"created_at"`
}

// MessagePeer 是会话对方的精简信息（用于聊天窗顶部）。
type MessagePeer struct {
	PeerID     uint64 `json:"peer_id"`
	PeerName   string `json:"peer_name"`
	PeerAvatar string `json:"peer_avatar"`
}

// ConversationItem 是会话列表里单个会话的响应视图。
type ConversationItem struct {
	PeerID      uint64    `json:"peer_id"`
	PeerName    string    `json:"peer_name"`
	PeerAvatar  string    `json:"peer_avatar"`
	LastMessage string    `json:"last_message"`
	LastAt      time.Time `json:"last_at"`
	LastMine    bool      `json:"last_mine"`    // 最后一条是否我发的
	UnreadCount int64     `json:"unread_count"` // 对方发给我、我还没读的条数
}
