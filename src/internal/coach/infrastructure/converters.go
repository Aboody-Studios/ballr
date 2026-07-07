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

func FromChatDomainToInfra(chatMsgDomain domain.ChatMessage) ChatMessage {
	return ChatMessage{
		ID: chatMsgDomain.ID,
		UserID: chatMsgDomain.UserID,
		Role: chatMsgDomain.Role,
		Content: chatMsgDomain.Content,
		CreatedAt: chatMsgDomain.CreatedAt,
	}
}
