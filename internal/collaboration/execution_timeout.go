package collaboration

const (
	DefaultCollabExecutionTimeoutSeconds     = 180
	DefaultCollabFileExecutionTimeoutSeconds = 300
)

// ExecutionTimeoutSeconds returns the generation deadline for a collaboration task.
func ExecutionTimeoutSeconds(task CollaborationTask, overrideSeconds int) int {
	if TaskRequiresFileDeliverable(task) {
		if overrideSeconds > 0 {
			return overrideSeconds
		}
		return DefaultCollabFileExecutionTimeoutSeconds
	}
	return DefaultCollabExecutionTimeoutSeconds
}
