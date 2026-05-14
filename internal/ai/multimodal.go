package ai

import (
	"context"

	"github.com/camronwood/neural-junkie/internal/protocol"
)

// MultimodalProvider is implemented by AI backends that can attach user images
// to the current user turn (conversation history remains text-only in v1).
type MultimodalProvider interface {
	GenerateMultimodal(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (string, error)
	GenerateMultimodalStream(ctx context.Context, prompt string, images []protocol.UserImagePart, conversationHistory []protocol.Message) (<-chan StreamToken, error)
}
