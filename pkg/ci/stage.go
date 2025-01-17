package ci

import (
	"context"
	"fmt"
	"github.com/hashicorp/hcl/v2"
)

const StageContextChildStatuses = "child_statuses"

func (s *Stage) Description() string {
	// TODO: implement
	return ""
}

func (s *Stage) Identifier() string {
	return fmt.Sprintf("%s.%s", StageBlock, s.Id)
}

func (s *Stage) Set(k any, v any) {
	if s.ctxInitialised == false {
		s.ctx = context.Background()
		s.ctxInitialised = true
	}
	s.ctx = context.WithValue(s.ctx, k, v)
}

func (s *Stage) Get(k any) any {
	if s.ctxInitialised {
		return s.ctx.Value(k)
	}
	return nil
}

func (s *Stage) Type() string {
	return StageBlock
}

func (s *Stage) IsDaemon() bool {
	return s.Daemon != nil && s.Daemon.Enabled
}

func (s Stages) ById(id string) (*Stage, hcl.Diagnostics) {
	for _, stage := range s {
		if stage.Id == id {
			return &stage, nil
		}
	}
	return nil, hcl.Diagnostics{
		{
			Severity: hcl.DiagError,
			Summary:  "Stage not found",
			Detail:   fmt.Sprintf("Stage with id %s not found", id),
		},
	}
}
