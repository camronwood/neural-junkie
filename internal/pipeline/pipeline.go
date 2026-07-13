package pipeline

import "context"

// Step is one named stage in a pipeline.
type Step interface {
	Name() string
	Run(ctx context.Context) error
}

// FuncStep wraps a function as a Step.
type FuncStep struct {
	StepName string
	Fn       func(ctx context.Context) error
}

func (s FuncStep) Name() string { return s.StepName }

func (s FuncStep) Run(ctx context.Context) error {
	if s.Fn == nil {
		return nil
	}
	return s.Fn(ctx)
}

// Run executes steps in order; stops on first error.
func Run(ctx context.Context, steps []Step) error {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if err := step.Run(ctx); err != nil {
			return err
		}
	}
	return nil
}

// RunWithHooks executes steps with optional before/after hooks (used in tests).
func RunWithHooks(ctx context.Context, steps []Step, before, after func(name string)) error {
	for _, step := range steps {
		if step == nil {
			continue
		}
		if before != nil {
			before(step.Name())
		}
		err := step.Run(ctx)
		if after != nil {
			after(step.Name())
		}
		if err != nil {
			return err
		}
	}
	return nil
}
