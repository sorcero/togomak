package ci

import (
	"context"
	"github.com/hashicorp/hcl/v2"
)

func (l *Local) Lifecycle(ctx context.Context) (*DaemonLifecycle, hcl.Diagnostics) {
	return nil, nil
}
