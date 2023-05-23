package utils

import (
	"runtime"
	"sync"
)

type ProcessGroup[I, O any] struct {
	proc  func(I) (O, error)
	abort chan struct{}
	wg    sync.WaitGroup

	Input  chan I
	Output chan O
	Err    chan error
}

func NewProcessGroup[I, O any](proc func(I) (O, error)) ProcessGroup[I, O] {
	routines := Max(Min(runtime.GOMAXPROCS(-1), runtime.NumCPU())-2, 1)
	routines = 1

	group := ProcessGroup[I, O]{
		proc:  proc,
		abort: make(chan struct{}),

		Input:  make(chan I, routines*4),
		Output: make(chan O, routines*4),
		Err:    make(chan error),
	}

	for i := 0; i < routines; i++ {
		group.wg.Add(1)
		go group.runProcessor()
	}

	go func() {
		group.wg.Wait()
		close(group.Output)
		close(group.Err)
	}()

	return group
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

func (g *ProcessGroup[I, O]) Close() {
	close(g.abort)
	close(g.Input)
	close(g.Output)
	close(g.Err)
	g.wg.Wait()
}
