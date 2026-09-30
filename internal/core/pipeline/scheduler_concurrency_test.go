package pipeline

import (
	"testing"
	"time"
)

func TestSchedulerChangesConcurrency(t *testing.T) {
	entered := make(chan string, 8)
	release := make(chan struct{}, 8)
	s := NewScheduler(NewCore(Config{}, nil, nil, nil), func(id string, err error) { entered <- id; <-release })
	if err := s.SetWorkers(1); err != nil {
		t.Fatal(err)
	}
	s.Start()
	t.Cleanup(func() {
		for range 8 {
			release <- struct{}{}
		}
		s.Stop()
	})
	for _, id := range []string{"a", "b", "c", "d"} {
		if err := s.Submit(deletionJob(t, id)); err != nil {
			t.Fatal(err)
		}
	}
	take := func(want string) {
		t.Helper()
		select {
		case got := <-entered:
			if got != want {
				t.Fatalf("got %s want %s", got, want)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("任务未启动")
		}
	}
	quiet := func() {
		t.Helper()
		select {
		case got := <-entered:
			t.Fatalf("超出并发限制: %s", got)
		case <-time.After(50 * time.Millisecond):
		}
	}
	take("a")
	quiet()
	if err := s.SetWorkers(2); err != nil {
		t.Fatal(err)
	}
	take("b")
	if err := s.SetWorkers(1); err != nil {
		t.Fatal(err)
	}
	release <- struct{}{}
	quiet()
	release <- struct{}{}
	take("c")
	quiet()
	release <- struct{}{}
	take("d")
}

func TestSchedulerRejectsInvalidWorkers(t *testing.T) {
	s := NewScheduler(nil, nil)
	for _, n := range []int{-1, 0, 9} {
		if err := s.SetWorkers(n); err == nil {
			t.Fatalf("接受无效并发数 %d", n)
		}
	}
	s.Stop()
	if err := s.Submit(Job{}); err == nil {
		t.Fatal("停止后仍可提交")
	}
	s.Stop()
}
