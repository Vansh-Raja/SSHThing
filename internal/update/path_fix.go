package update

import "context"

func FixPathConflicts(ctx context.Context, desiredExe string) (PathHealth, error) {
	return fixPathConflicts(ctx, desiredExe)
}
