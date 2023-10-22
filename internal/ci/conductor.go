package ci

import (
	"context"
	"github.com/google/uuid"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/sirupsen/logrus"
	"github.com/srevinsaju/togomak/v1/internal/global"
	"github.com/srevinsaju/togomak/v1/internal/meta"
	"github.com/srevinsaju/togomak/v1/internal/x"
	"os"
	"path/filepath"
	"time"
)

type ConductorOption func(*Conductor)

func ConductorWithLogger(logger logrus.Ext1FieldLogger) ConductorOption {
	return func(c *Conductor) {
		c.RootLogger = logger
	}
}

func ConductorWithConfig(cfg ConductorConfig) ConductorOption {
	return func(c *Conductor) {
		c.Config = cfg
	}
}

func ConductorWithContext(ctx context.Context) ConductorOption {
	return func(c *Conductor) {
		c.ctx = ctx
	}
}

func ConductorWithParser(parser *hclparse.Parser) ConductorOption {
	return func(c *Conductor) {
		c.Parser = parser
	}
}

func ConductorWithDiagWriter(diagWriter hcl.DiagnosticWriter) ConductorOption {
	return func(c *Conductor) {
		c.DiagWriter = diagWriter
	}
}

func ConductorWithEvalContext(evalContext *hcl.EvalContext) ConductorOption {
	return func(c *Conductor) {
		c.EvalContext = evalContext
	}
}

func ConductorWithProcess(process Process) ConductorOption {
	return func(c *Conductor) {
		c.Process = process
	}
}

type Conductor struct {
	RootLogger logrus.Ext1FieldLogger
	Config     ConductorConfig
	ctx        context.Context

	// Process is the current process
	Process Process

	// hcl stuff

	// Parser is the HCL parser
	Parser *hclparse.Parser

	// DiagWriter is the HCL diagnostic writer, it is used to write the diagnostics
	// to os.Stdout
	DiagWriter hcl.DiagnosticWriter

	// EvalContext is the HCL evaluation context
	EvalContext *hcl.EvalContext

	parent *Conductor
}

func (c *Conductor) Child(opts ...ConductorOption) *Conductor {
	inheritOpts := []ConductorOption{
		ConductorWithConfig(c.Config),
	}
	opts = append(inheritOpts, opts...)
	child := NewConductor(c.Config, opts...)
	child.parent = c
	return child
}

func (c *Conductor) Parent() *Conductor {
	return c.parent
}

func (c *Conductor) RootParent() *Conductor {
	if c.parent == nil {
		return c
	}
	return c.parent.RootParent()
}

func (c *Conductor) Logger() logrus.Ext1FieldLogger {
	return c.RootLogger
}

type Process struct {
	Id uuid.UUID

	Executable string

	// BootTime is the time when the process was started
	BootTime time.Time

	// TempDir is the temporary directory created for the process
	TempDir string
}

func (c *Conductor) Context() context.Context {
	return c.ctx
}

func NewProcess(cfg ConductorConfig) Process {
	e, err := os.Executable()
	x.Must(err)

	pipelineId := uuid.New()

	// create a temporary directory
	tempDir, err := os.MkdirTemp("", "togomak")
	x.Must(err)
	global.SetTempDir(tempDir)

	return Process{
		Id:         pipelineId,
		Executable: e,
		BootTime:   time.Now(),
		TempDir:    tempDir,
	}
}

func Chdir(cfg ConductorConfig, logger *logrus.Logger) string {
	cwd := cfg.Paths.Cwd
	if cwd == "" {
		cwd = filepath.Dir(cfg.Paths.Pipeline)
		if filepath.Base(cwd) == meta.BuildDirPrefix {
			cwd = filepath.Dir(cwd)
		}
	}
	err := os.Chdir(cwd)
	if err != nil {
		logger.Fatal(err)
	}
	cwd, err = os.Getwd()
	x.Must(err)
	logger.Debug("changing working directory to ", cwd)
	return cwd

}

func NewConductor(cfg ConductorConfig, opts ...ConductorOption) *Conductor {
	parser := hclparse.NewParser()

	diagWriter := hcl.NewDiagnosticTextWriter(os.Stdout, parser.Files(), 0, true)

	logger := NewLogger(cfg)
	global.SetLogger(logger)

	dir := Chdir(cfg, logger)
	cfg.Paths.Cwd = dir

	if cfg.Paths.Module == "" {
		cfg.Paths.Module = cfg.Paths.Cwd
	}

	process := NewProcess(cfg)

	c := &Conductor{
		Parser:     parser,
		DiagWriter: diagWriter,
		ctx:        context.Background(),

		Process: process,

		RootLogger: logger,
		Config:     cfg,

		EvalContext: CreateEvalContext(cfg, process),
	}
	for _, opt := range opts {
		opt(c)
	}

	if !c.Config.Behavior.Child.Enabled {
		logger.Infof("%s (version=%s)", meta.AppName, meta.AppVersion)
	}

	return c
}

func (c *Conductor) Destroy() {
	c.Logger().Debug("removing temporary directory")
	err := os.RemoveAll(c.Process.TempDir)
	if err != nil {
		c.Logger().Warnf("failed to remove temporary directory: %s", err)
	}

	c.Logger().Debug("destroying togomak")

	c.RootLogger = nil
	c.Config = ConductorConfig{}
	c.Parser = nil
	c.DiagWriter = nil
}

func (c *Conductor) Update(opts ...ConductorOption) {
	for _, opt := range opts {
		opt(c)
	}
}
