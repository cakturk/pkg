package freq

import (
	"time"

	"periph.io/x/periph/conn/physic"
)

type Counter struct {
	start, finish time.Time
	count         int64
}

func (f *Counter) Cycle() {
	f.count++
}

func (f *Counter) Start() {
	f.start = time.Now()
}

func (f *Counter) Stop() {
	f.finish = time.Now()
}

func (f *Counter) Frequency() physic.Frequency {
	dur := f.finish.Sub(f.start)
	return physic.Frequency(
		float64(f.count) / dur.Seconds() * float64(physic.Hertz),
	)
}

func StartNewCounter() *Counter {
	return &Counter{
		start: time.Now(),
	}
}
