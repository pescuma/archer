package utils

import (
	"runtime"
	"sync"
)

type ParallelOptions struct {
	Routines     int
	InputFactor  int
	OutputFactor int
}

func ParallelFor[T, O any](col []T, proc func(T) (O, error), opts ...ParallelOptions) *ProcessGroup[T, O] {
	group := NewProcessGroup(proc, opts...)

	go func() {
		for _, w := range col {
			group.Input <- w
		}

		group.FinishedInput()
	}()

	return group
}

type ProcessGroup[I, O any] struct {
	proc  func(I) (O, error)
	abort chan struct{}
	wg    sync.WaitGroup

	Input  chan I
	Output chan O
	Err    chan error
}

func NewProcessGroup[I, O any](proc func(I) (O, error), opts ...ParallelOptions) *ProcessGroup[I, O] {
	o := ParallelOptions{
		Routines:     Max(Min(runtime.GOMAXPROCS(-1), runtime.NumCPU()/2)-1, 1),
		InputFactor:  2,
		OutputFactor: 2,
	}
	for _, oi := range opts {
		if oi.Routines > 0 {
			o.Routines = oi.Routines
		}
		if oi.InputFactor > 0 {
			o.InputFactor = oi.InputFactor
		}
		if oi.OutputFactor > 0 {
			o.OutputFactor = oi.OutputFactor
		}
	}

	group := ProcessGroup[I, O]{
		proc:  proc,
		abort: make(chan struct{}),

		Input:  make(chan I, o.InputFactor*o.Routines),
		Output: make(chan O, o.OutputFactor*o.Routines),
		Err:    make(chan error),
	}

	for i := 0; i < o.Routines; i++ {
		group.wg.Add(1)
		go group.runProcessor()
	}

	go func() {
		group.wg.Wait()
		close(group.Output)
		close(group.Err)
	}()

	return &group
}

func (g *ProcessGroup[I, O]) runProcessor() {
	defer g.wg.Done()

	for {
		select {
		case <-g.abort:
			return

		case input, ok := <-g.Input:
			if !ok {
				return
			}

			output, err := g.proc(input)
			if err != nil {
				g.Abort(err)
				return
			}

			g.Output <- output
		}
	}
}

func (g *ProcessGroup[I, O]) FinishedInput() {
	close(g.Input)
}

func (g *ProcessGroup[I, O]) Abort(err error) {
	g.Err <- err
	close(g.abort)
}

func (g *ProcessGroup[I, O]) Aborted() bool {
	select {
	case <-g.abort:
		return true
	default:
		return false
	}
}

func (g *ProcessGroup[I, O]) Error() error {
	return <-g.Err
}

func (g *ProcessGroup[I, O]) Close() {
	close(g.abort)
	close(g.Input)
	close(g.Output)
	close(g.Err)
	g.wg.Wait()
}
