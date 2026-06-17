package claudecode

import "github.com/google/uuid"

// NewSessionID returns a new random session ID for use with WithSessionID
// and WithResume. Sessions let you persist conversation context across
// separate agent.Run() calls or program restarts.
//
// In one-shot mode, pass the returned ID to WithSessionID on the first
// call and WithResume on subsequent calls to continue the conversation.
// In Client mode, sessions are maintained automatically by keeping the
// CLI process alive — no session ID is needed.
func NewSessionID() string {
	return uuid.New().String()
}
