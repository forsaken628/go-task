package task

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

var releaseMode = false

type SubTaskAdder func(tk *Task, async bool)

type Maker interface {
	Inject(adder SubTaskAdder) func(context.Context) error
}

type Task struct {
	Name         string
	subTask      []*Task
	asyncSubTask []*Task
	// 当前任务需完成的任务
	fn func(context.Context) error

	done chan struct{}
	mu   sync.Mutex
}

func NewTask(tm Maker) *Task {
	tk := &Task{
		done: make(chan struct{}),
	}
	fn := tm.Inject(func(t *Task, async bool) {
		if !releaseMode {
			for _, tk0 := range tk.subTask {
				if tk0 == t {
					panic("same task add twice")
				}
			}
			for _, tk0 := range tk.asyncSubTask {
				if tk0 == t {
					panic("same task add twice")
				}
			}
		}
		if async {
			tk.asyncSubTask = append(tk.asyncSubTask, t)
			return
		}
		tk.subTask = append(tk.subTask, t)
	})
	tk.fn = fn
	return tk
}

func (t *Task) Run(pctx context.Context) error {
	defer close(t.done)
	select {
	case <-pctx.Done():
		return pctx.Err()
	default:
	}

	ctx, cancel := context.WithCancel(pctx)
	g, ctx := errgroup.WithContext(ctx)

	for _, st := range t.asyncSubTask {
		st := st
		g.Go(func() (err error) {
			if releaseMode {
				defer func() {
					switch r := recover().(type) {
					case error:
						err = r
					default:
						err = errors.New(fmt.Sprintf("[PANIC] %#v", r))
					}
				}()
			}

			return st.Run(ctx)
		})
	}

	for _, st := range t.subTask {
		err := st.Run(ctx)
		if err != nil {
			cancel()
			return err
		}
	}

	var err error
	if len(t.asyncSubTask) != 0 {
		err = g.Wait()
		if err != nil {
			return err
		}
	}
	return t.fn(ctx)
}

func (t *Task) Done() <-chan struct{} {
	t.mu.Lock()
	if t.done == nil {
		t.done = make(chan struct{})
	}
	d := t.done
	t.mu.Unlock()
	return d
}

type FuncInjectMaker func(adder SubTaskAdder) func(context.Context) error

func (f FuncInjectMaker) Inject(adder SubTaskAdder) func(ctx context.Context) error {
	return f(adder)
}

func (f FuncInjectMaker) Task() *Task {
	return NewTask(f)
}

type FuncMaker func(context.Context) error

func (f FuncMaker) Inject(SubTaskAdder) func(context.Context) error {
	return f
}

func (f FuncMaker) Task() *Task {
	return NewTask(f)
}

func Timeout(timeout time.Duration, task *Task) *Task {
	return FuncMaker(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return task.Run(ctx)
	}).Task()
}
