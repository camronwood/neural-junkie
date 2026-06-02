package collaboration

// UpstreamTasksComplete reports whether every explicit dependency for task is satisfied.
func UpstreamTasksComplete(task CollaborationTask, all []CollaborationTask, policy BlockedUpstreamPolicy) bool {
	if len(task.Dependencies) == 0 && len(task.DependencyEdges) == 0 && len(task.DependencyGroups) == 0 {
		return true
	}
	c := &Collaboration{
		Tasks:           all,
		ExecutionPolicy: ExecutionPolicy{BlockedUpstreamPolicy: policy},
	}
	probe := task
	probe.Status = TaskPending
	return IsTaskReadyForCollab(probe, c)
}
