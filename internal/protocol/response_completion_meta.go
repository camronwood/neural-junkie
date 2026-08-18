package protocol

// Response completion metadata keys on assistant chat replies.
const (
	MetadataTerminalReason         = "terminal_reason"
	MetadataProviderTerminalReason = "provider_terminal_reason"
	MetadataContinuationAvailable  = "continuation_available"
	MetadataContinuationOf         = "continuation_of"
	MetadataContinuationReason     = "continuation_reason"
)

// Terminal reason values (provider-neutral).
const (
	TerminalReasonStop      = "stop"
	TerminalReasonLength    = "length"
	TerminalReasonToolCalls = "tool_calls"
	TerminalReasonTimeout   = "timeout"
	TerminalReasonCancelled = "cancelled"
	TerminalReasonError     = "error"
)

// ApplyResponseCompletionMetadata stamps length/timeout completion fields on msg.
func ApplyResponseCompletionMetadata(msg *Message, terminalReason, providerReason string) {
	if msg == nil {
		return
	}
	reason := terminalReason
	if reason == "" {
		return
	}
	if msg.Metadata == nil {
		msg.Metadata = make(map[string]interface{})
	}
	msg.Metadata[MetadataTerminalReason] = reason
	if providerReason != "" {
		msg.Metadata[MetadataProviderTerminalReason] = providerReason
	}
	if reason == TerminalReasonLength {
		msg.Metadata[MetadataContinuationAvailable] = true
	}
}
