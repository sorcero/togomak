package ci

import (
	"github.com/hashicorp/hcl/v2"
	"github.com/kendru/darwin/go/depgraph"
	"github.com/sirupsen/logrus"
	"github.com/srevinsaju/togomak/v1/internal/blocks"
	"github.com/srevinsaju/togomak/v1/internal/rules"
	"github.com/srevinsaju/togomak/v1/internal/runnable"
	"time"
)

func BlockRunWithRetries(conductor *Conductor, runnableId string, runnable Block, handler *Handler, togomakLogger logrus.Ext1FieldLogger, opts ...runnable.Option) {
	logger := togomakLogger.WithField("orchestra", "run")
	logger.Debug("starting runnable with retries ", runnableId)
	stageDiags := runnable.Run(conductor, opts...)

	handler.Tracker.AppendCompleted(runnable)
	logger.Tracef("signaling runnable %s", runnableId)

	if !stageDiags.HasErrors() {
		if runnable.IsDaemon() {
			handler.Tracker.DaemonDone()
		} else {
			handler.Tracker.RunnableDone()
		}
		return
	}
	if !runnable.CanRetry() {
		logger.Debug("runnable cannot be retried")
	} else {
		logger.Infof("retrying runnable %s", runnableId)
		retryCount := 0
		retryMinBackOff := time.Duration(runnable.MinRetryBackoff()) * time.Second
		retryMaxBackOff := time.Duration(runnable.MaxRetryBackoff()) * time.Second
		retrySuccess := false
		for retryCount < runnable.MaxRetries() {
			retryCount++
			sleepDuration := time.Duration(1) * time.Second
			if runnable.RetryExponentialBackoff() {

				if retryMinBackOff*time.Duration(retryCount) > retryMaxBackOff && retryMaxBackOff > 0 {
					sleepDuration = retryMaxBackOff
				} else {
					sleepDuration = retryMinBackOff * time.Duration(retryCount)
				}
			} else {
				sleepDuration = retryMinBackOff
			}
			logger.Warnf("runnable %s failed, retrying in %s", runnableId, sleepDuration)
			time.Sleep(sleepDuration)
			sDiags := runnable.Run(conductor, opts...)
			stageDiags = append(stageDiags, sDiags...)

			if !sDiags.HasErrors() {
				retrySuccess = true
				break
			}
		}

		if !retrySuccess {
			logger.Warnf("runnable %s failed after %d retries", runnableId, retryCount)
		}

	}
	handler.Diags.Extend(stageDiags)
	if runnable.IsDaemon() {
		handler.Tracker.DaemonDone()
	} else {
		handler.Tracker.RunnableDone()
	}
}

func BlockCanRun(runnable Block, conductor *Conductor, filterList rules.Operations, filterQuery QueryEngines, runnableId string, depGraph *depgraph.Graph, opts ...runnable.Option) (ok bool, overridden bool, diags hcl.Diagnostics) {

	ok, d := runnable.CanRun(conductor, opts...)
	if d.HasErrors() {
		diags = diags.Extend(d)
		return false, false, diags
	}

	if runnable.Type() != blocks.StageBlock && runnable.Type() != blocks.ModuleBlock {
		// TODO: optimize, PipelineRun only required data blocks
		return ok, false, diags
	}

	runnable.Set(StageContextChildStatuses, filterList.Children(runnableId).Marshall())

	if (runnable.Type() == blocks.StageBlock || runnable.Type() == blocks.ModuleBlock) && len(filterQuery) != 0 {
		ok, overridden, d = filterQuery.Eval(conductor, ok, runnable.(PhasedBlock))
		if d.HasErrors() {
			diags = diags.Extend(d)
			return false, false, diags
		}
	}

	if len(filterList) == 0 {
		filterList = append(filterList, rules.NewOperation(rules.OperationTypeAnd, "default"))
	}
	runnable.Set(StageContextChildStatuses, filterList.Children(runnableId).Marshall())

	for _, rule := range filterList {
		if rule.RunnableId() == "all" {
			return ok, false, diags
		}
	}

	oldOk := ok
	ok = false
	overridden = false

	for _, rule := range filterList {
		if rule.RunnableId() == runnableId && rule.Operation() == rules.OperationTypeAdd {
			ok = true
			overridden = true
		}
		if rule.RunnableId() == runnableId && rule.Operation() == rules.OperationTypeSub {
			ok = false
			overridden = true
		}
		if rule.RunnableId() == runnableId && rule.Operation() == rules.OperationTypeAnd {
			ok = oldOk
			overridden = true
		}
		if rule.Operation() == rules.OperationTypeAnd && depGraph.DependsOn(rule.RunnableId(), runnableId) {
			ok = oldOk
			overridden = true
		}
		if runnable.Type() == blocks.StageBlock || runnable.Type() == blocks.ModuleBlock {
			stage := runnable.(PhasedBlock)
			if stage.LifecycleConfig() != nil {
				evalContext := conductor.Eval().Context()
				conductor.Eval().Mutex().RLock()
				phaseHcl, d := stage.LifecycleConfig().Phase.Value(evalContext)
				conductor.Eval().Mutex().RUnlock()

				if d.HasErrors() {
					diags = diags.Extend(d)
					return false, false, diags
				}
				phases := phaseHcl.AsValueSlice()

				for _, phase := range phases {
					if rule.RunnableId() == phase.AsString() {
						overridden = false
						ok = oldOk
					}
				}
				if len(phases) == 0 && rule.RunnableId() == "default" {
					ok = oldOk
					overridden = false
				}
			} else {
				if rule.RunnableId() == "default" {
					ok = oldOk
					overridden = false
				}
			}
		}
	}
	return ok, overridden, diags
}