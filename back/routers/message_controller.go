package routers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"warehouse-web/middlewares"
	"warehouse-web/models"
	"warehouse-web/services"
	"warehouse-web/utils"
)

// MessageController 处理用户私信相关请求（均需登录）。
type MessageController struct {
	service *services.MessageService
}

// NewMessageController 创建 MessageController。
func NewMessageController(service *services.MessageService) *MessageController {
	return &MessageController{service: service}
}

// RegisterRoutes 将私信路由绑定到指定路由组（该组应已挂载 AuthRequired 中间件）。
//
//	POST /api/messages                       body: { receiver_id, content }
//	GET  /api/messages/conversations         我的会话列表
//	GET  /api/messages/unread-count          未读总数（导航栏小红点）
//	GET  /api/messages/with/:userId          与某人的聊天记录（打开即标记已读）
func (ctl *MessageController) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("", ctl.Send)
	group.GET("/conversations", ctl.ListConversations)
	group.GET("/unread-count", ctl.UnreadCount)
	group.GET("/with/:userId", ctl.Thread)
}

type sendMessageRequest struct {
	ReceiverID uint64 `json:"receiver_id" binding:"required"`
	Content    string `json:"content"     binding:"required"`
}

// Send 发送一条私信。
func (ctl *MessageController) Send(c *gin.Context) {
	var req sendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "请求体格式不正确", err.Error())
		return
	}
	me := ctl.currentUserID(c)
	msg, err := ctl.service.SendMessage(me, req.ReceiverID, req.Content)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, ctl.toItem(me, *msg))
}

// ListConversations 返回当前用户的会话列表。
func (ctl *MessageController) ListConversations(c *gin.Context) {
	me := ctl.currentUserID(c)
	items, err := ctl.service.ListConversations(me)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	for i := range items {
		items[i].PeerAvatar = toPublicURL(c, items[i].PeerAvatar)
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{"items": items})
}

// UnreadCount 返回当前用户的未读私信总数。
func (ctl *MessageController) UnreadCount(c *gin.Context) {
	me := ctl.currentUserID(c)
	n, err := ctl.service.CountUnread(me)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{"count": n})
}

// Thread 返回与某人的聊天记录，并把对方发来的未读消息标记为已读。
func (ctl *MessageController) Thread(c *gin.Context) {
	peerID, err := strconv.ParseUint(c.Param("userId"), 10, 64)
	if err != nil || peerID == 0 {
		utils.JSONError(c, http.StatusBadRequest, "invalid_request", "userId 必须为正整数")
		return
	}
	me := ctl.currentUserID(c)
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	peer, msgs, err := ctl.service.GetThread(me, peerID, limit)
	if err != nil {
		ctl.handleError(c, err)
		return
	}
	peer.PeerAvatar = toPublicURL(c, peer.PeerAvatar)

	items := make([]models.MessageItem, 0, len(msgs))
	for _, m := range msgs {
		items = append(items, ctl.toItem(me, m))
	}
	utils.JSONSuccess(c, http.StatusOK, gin.H{"peer": peer, "items": items})
}

// toItem 把持久化记录转成响应视图（带 Mine / Read 标记）。
func (ctl *MessageController) toItem(meID uint64, m models.Message) models.MessageItem {
	return models.MessageItem{
		ID:         m.ID,
		SenderID:   m.SenderID,
		ReceiverID: m.ReceiverID,
		Content:    m.Content,
		Mine:       m.SenderID == meID,
		Read:       m.ReadAt != nil,
		CreatedAt:  m.CreatedAt,
	}
}

// currentUserID 从 gin.Context 中取出当前登录用户的 ID。
func (ctl *MessageController) currentUserID(c *gin.Context) models.PrimaryKey {
	val, _ := c.Get(middlewares.CtxKeyUserID)
	return val.(models.PrimaryKey)
}

// handleError 统一处理服务层错误。
func (ctl *MessageController) handleError(c *gin.Context, err error) {
	if svcErr, ok := err.(*services.ServiceError); ok {
		utils.JSONError(c, svcErr.Status, svcErr.Code, svcErr.Message)
		return
	}
	utils.JSONError(c, http.StatusInternalServerError, "internal_error", "服务器开小差了，请稍后重试")
}
