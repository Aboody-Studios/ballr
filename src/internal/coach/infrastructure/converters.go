package infrastructure

import "github.com/Aboody-Studios/ballr/src/internal/coach/domain"

func FromChatInfraToDomain(chatMsgInfra ChatMessage) domain.ChatMessage {
	return domain.ChatMessage{
		ID:        chatMsgInfra.ID,
		UserID:    chatMsgInfra.UserID,
		Role:      chatMsgInfra.Role,
		Content:   chatMsgInfra.Content,
		CreatedAt: chatMsgInfra.CreatedAt,
	}
}
