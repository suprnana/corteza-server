package types

import (
	"context"
	"fmt"
	"time"

	"github.com/cortezaproject/corteza-server/pkg/expr"
	"github.com/cortezaproject/corteza-server/pkg/logger"
	"github.com/cortezaproject/corteza-server/pkg/wfexec"
	"go.uber.org/zap"
)

type (
	FunctionHandler func(ctx context.Context, in *expr.Vars) (*expr.Vars, error)
	IteratorHandler func(ctx context.Context, in *expr.Vars) (wfexec.IteratorHandler, error)

	// workflow functions are defined in the core code and through plugins
	Function struct {
		Ref        string        `json:"ref,omitempty"`
		Kind       string        `json:"kind,omitempty"`
		Meta       *FunctionMeta `json:"meta,omitempty"`
		Parameters ParamSet      `json:"parameters,omitempty"`
		Results    ParamSet      `json:"results,omitempty"`

		Handler  FunctionHandler `json:"-"`
		Iterator IteratorHandler `json:"-"`

		Labels   map[string]string `json:"labels,omitempty"`
		Disabled bool              `json:"disabled,omitempty"`
	}

	FunctionMeta struct {
		Short       string                 `json:"short,omitempty"`
		Description string                 `json:"description,omitempty"`
		Visual      map[string]interface{} `json:"visual,omitempty"`
	}

	functionStep struct {
		wfexec.StepIdentifier
		def       *Function
		arguments ExprSet
		results   ExprSet
	}

	iteratorStep struct {
		wfexec.StepIdentifier
		def       *Function
		arguments ExprSet
		results   ExprSet
		next      wfexec.Step
		exit      wfexec.Step
	}
)

const (
	FunctionKindFunction = "function"
	FunctionKindIterator = "iterator"
)

// FunctionStep initializes new function step with function definition and configured arguments and results
func FunctionStep(def *Function, arguments, results ExprSet) (*functionStep, error) {
	if def.Kind != FunctionKindFunction {
		return nil, fmt.Errorf("expecting function kind")
	}

	return &functionStep{def: def, arguments: arguments, results: results}, nil
}

// Exec executes function step
//
// Configured arguments are evaluated with the step's scope and input and passed
// to the function
//
// Configured results are evaluated with the results from the function and final scope is then returned
func (f functionStep) Exec(ctx context.Context, r *wfexec.ExecRequest) (wfexec.ExecResponse, error) {
	var (
		started       = time.Now()
		args, results *expr.Vars
		err           error

		log = logger.ContextValue(ctx, zap.NewNop()).With(
			zap.String("functionRef", f.def.Ref),
			zap.String("functionKind", "function"),
		)
	)

	defer func() {
		log := log.With(zap.Duration("execTime", time.Now().Sub(started)))

		if err == nil {
			log.Debug("executed")
		} else {
			log.Warn("executed with errors", zap.Error(err))
		}
	}()

	ctx = logger.ContextWithValue(ctx, log)

	if len(f.arguments) > 0 {
		// Arguments defined, get values from scope and use them when calling
		// function/handler
		args, err = f.arguments.Eval(ctx, r.Scope.MustMerge(r.Input))
		if err != nil {
			return nil, err
		}
	}

	results, err = f.def.Handler(ctx, args)
	if err != nil {
		return nil, err
	}

	if len(f.results) == 0 {
		// No results defined, nothing to return
		return expr.NewVars(nil)
	}

	results, err = f.results.Eval(ctx, results)
	if err != nil {
		return nil, err
	}

	return results, nil
}

// IteratorStep initializes new iterator step with function (iterator) and other parameters
//
// Pointers to next and exit steps are are given and then passed on to the iterator
func IteratorStep(def *Function, arguments, results ExprSet, next, exit wfexec.Step) (*iteratorStep, error) {
	if def.Kind != FunctionKindIterator {
		return nil, fmt.Errorf("expecting iterator kind")
	}

	return &iteratorStep{
		def:       def,
		arguments: arguments,
		results:   results,
		next:      next,
		exit:      exit,
	}, nil
}

// Exec configures and returns generic iterator
//
// This fn is executed only once, before 1st iteration. From there on, session executor uses Iterator to
// iterate over all items, elements or content that iterator serves it through Next() fn
//
// See also EvalResults
func (f *iteratorStep) Exec(ctx context.Context, r *wfexec.ExecRequest) (wfexec.ExecResponse, error) {
	var (
		started = time.Now()
		args    *expr.Vars
		err     error
		ih      wfexec.IteratorHandler

		log = logger.ContextValue(ctx, zap.NewNop()).With(
			zap.String("functionRef", f.def.Ref),
			zap.String("functionKind", "iterator"),
		)
	)

	defer func() {
		log := log.With(zap.Duration("execTime", time.Now().Sub(started)))

		if err == nil {
			log.Debug("executed")
		} else {
			log.Warn("executed with errors", zap.Error(err))
		}
	}()

	if len(f.arguments) > 0 {
		// Arguments defined, get values from scope and use them when calling
		// iterator/handler
		args, err = f.arguments.Eval(ctx, r.Scope.MustMerge(r.Input))
		if err != nil {
			return nil, err
		}
	}

	if ih, err = f.def.Iterator(ctx, args); err != nil {
		return nil, err
	}

	return wfexec.GenericIterator(f, f.next, f.exit, ih), nil
}

// EvalResults is called from iterator's Next() fn.
//
// It prepares scope for each iteration by evaluating results from the iterator function
func (f *iteratorStep) EvalResults(ctx context.Context, results *expr.Vars) (*expr.Vars, error) {
	if results.Len() == 0 || len(f.results) == 0 {
		// No results or result expressions defined, nothing to return
		return &expr.Vars{}, nil
	}

	return f.results.Eval(ctx, results)
}
