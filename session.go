package claudecode

import "github.com/google/uuid"

// NewSessionID returns a new random UUID for use with [WithSessionID] and
// [WithResume]. Sessions persist conversation context across separate
// [ClaudeCodeAgent.Run] calls or program restarts.
//
// Pass the returned ID to [WithSessionID] on the first call and
// [WithResume] on subsequent calls to continue the conversation.
// See [examples/session] for a complete example.
func NewSessionID() string {
	return uuid.New().String()
}
