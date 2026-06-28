package infrastructure

import (
	"context"

	"github.com/Aboody-Studios/ballr/src/internal/coach/domain"
	"google.golang.org/genai"
)

type GeminiCoach struct {
	genai.Client
}

// TODO!: Refactor ?
func (gc *GeminiCoach) GenerateResponse(ctx context.Context, msg string, chatMessages []domain.ChatMessage) (string, error) {
	var contents []*genai.Content

	for _, message := range chatMessages {
		var parts []*genai.Part

		part := genai.Part{
			Text: message.Content,
		}
		parts = append(parts, &part)

		content := genai.Content{
			Parts: parts,
			Role:  message.Role,
		}

		contents = append(contents, &content)
	}

	chatSession, err := gc.Chats.Create(ctx, "gemini-2.5-flash", nil, contents)
	if err != nil {
		return "", err
	}

	response, resErr := chatSession.SendMessage(ctx, genai.Part{Text: msg})
	if resErr != nil {
		return "", resErr
	}

	return response.Text(), nil
}
