package main

import (
	"fmt"
	"time"
)

type timing struct {
	Name     string
	Duration time.Duration
}

type timingSet struct {
	Timings []timing
}

func (t *timingSet) Start(name string) (end func()) {
	s := time.Now()
	return func() { t.Add(name, time.Since(s)) }
}

func (t *timingSet) Add(name string, dur time.Duration) {
	t.Timings = append(t.Timings, timing{name, dur})
}

func (t *timingSet) Total() time.Duration {
	var dur time.Duration
	for _, i := range t.Timings {
		dur += i.Duration
	}
	return dur
}

func (t *timingSet) String() string {
	items := make([]string, len(t.Timings))
	for i, v := range t.Timings {
		items[i] = fmt.Sprintf("%v: %-8v", v.Name, v.Duration.Truncate(time.Microsecond))
	}
	return fmt.Sprintf("%-8v %v", t.Total().Truncate(time.Microsecond), items)
}
