package uvgo

import (
	"context"
	"errors"
	"os"
	"sync"
	"syscall"
	"time"
)

// StatT holds the subset of file metadata compared by libuv’s statbuf_eq,
// using portable os.FileInfo fields. Inode/device and birth time are not
// filled on all platforms; see StatFromFileInfo.
type StatT struct {
	Size    int64
	Mode    uint32
	ModTime time.Time
}

// StatFromFileInfo fills StatT from os.FileInfo for use in comparisons and callbacks.
func StatFromFileInfo(fi os.FileInfo) StatT {
	if fi == nil {
		return StatT{}
	}
	return StatT{
		Size:    fi.Size(),
		Mode:    uint32(fi.Mode()),
		ModTime: fi.ModTime(),
	}
}

func statEqual(a, b *StatT) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Size == b.Size &&
		a.Mode == b.Mode &&
		a.ModTime.Equal(b.ModTime)
}

var zeroStat StatT

func errnoFromStatError(err error) int {
	if err == nil {
		return 0
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		var e syscall.Errno
		if errors.As(pathErr.Err, &e) {
			return int(e)
		}
	}
	var e syscall.Errno
	if errors.As(err, &e) {
		return int(e)
	}
	return -1
}

// FSPollCallback mirrors uv_fs_poll_cb: status is 0 on success, otherwise
// a rough errno-style value (negative path errors may surface as -1 on some OSes).
// On error, curr points at zeroStat; prev holds the last successfully stat’ed snapshot.
type FSPollCallback func(handle *FSPoll, status int, prev, curr *StatT)

// FSPoll is the analogue of uv_fs_poll_t (polling via repeated os.Stat).
type FSPoll struct {
	st *fspollState
}

type fspollState struct {
	mu sync.Mutex

	owner    *FSPoll
	path     string
	interval time.Duration
	cb       FSPollCallback

	active bool
	ctx    context.Context
	cancel context.CancelFunc

	loop *Loop

	// busy: 0 = no successful baseline yet; 1 = last stat ok; <0 = errno
	busy int
	prev StatT
}

type fspollJob struct {
	fp     *fspollState
	fi     os.FileInfo
	err    error
	start  time.Time
	resume chan<- time.Duration
}

// FSPollStart corresponds to uv_fs_poll_start: polls path every intervalMs
// milliseconds (minimum 1, matching libuv). Callbacks run on the Loop.Run goroutine.
func (l *Loop) FSPollStart(path string, intervalMs int, cb FSPollCallback) (*FSPoll, error) {
	if cb == nil {
		return nil, errors.New("uvgo: nil fs poll callback")
	}
	l.mu.Lock()
	stopped := l.stopped
	l.mu.Unlock()
	if stopped {
		return nil, ErrStopped
	}
	if intervalMs <= 0 {
		intervalMs = 1
	}

	out := &FSPoll{}
	st := &fspollState{
		owner:    out,
		path:     path,
		interval: time.Duration(intervalMs) * time.Millisecond,
		cb:       cb,
		active:   true,
		busy:     0,
		loop:     l,
	}
	out.st = st

	ctx, cancel := context.WithCancel(l.ctx)
	st.ctx = ctx
	st.cancel = cancel

	l.wg.Add(1)
	go l.fspollWorker(st)

	return out, nil
}

// Stop corresponds to uv_fs_poll_stop.
func (fp *FSPoll) Stop() {
	if fp == nil || fp.st == nil {
		return
	}
	st := fp.st
	st.mu.Lock()
	st.active = false
	cancel := st.cancel
	st.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// Path corresponds to uv_fs_poll_getpath when the watcher is still active.
func (fp *FSPoll) Path() (string, bool) {
	if fp == nil || fp.st == nil {
		return "", false
	}
	st := fp.st
	st.mu.Lock()
	defer st.mu.Unlock()
	if !st.active {
		return "", false
	}
	return st.path, true
}

func (l *Loop) fspollWorker(st *fspollState) {
	defer l.wg.Done()

	for {
		select {
		case <-st.ctx.Done():
			return
		default:
		}

		st.mu.Lock()
		active := st.active
		st.mu.Unlock()
		if !active {
			return
		}

		start := time.Now()
		fi, err := os.Stat(st.path)

		resume := make(chan time.Duration, 1)
		job := &fspollJob{
			fp:     st,
			fi:     fi,
			err:    err,
			start:  start,
			resume: resume,
		}

		select {
		case l.fspollQ <- job:
		case <-st.ctx.Done():
			return
		}

		var d time.Duration
		select {
		case d = <-resume:
		case <-st.ctx.Done():
			return
		}
		if d < 0 {
			d = 0
		}

		select {
		case <-time.After(d):
		case <-st.ctx.Done():
			return
		}
	}
}

func (l *Loop) fspollDispatch(j *fspollJob) {
	fp := j.fp

	fp.mu.Lock()
	if !fp.active {
		fp.mu.Unlock()
		j.resume <- 0
		return
	}
	fp.mu.Unlock()

	if j.err != nil {
		status := errnoFromStatError(j.err)
		fp.mu.Lock()
		if fp.busy != status {
			prev := fp.prev
			fp.busy = status
			fp.mu.Unlock()
			fp.cb(fp.owner, status, &prev, &zeroStat)
		} else {
			fp.mu.Unlock()
		}
		d := fp.interval - (time.Since(j.start) % fp.interval)
		j.resume <- d
		return
	}

	curr := StatFromFileInfo(j.fi)

	fp.mu.Lock()
	busy := fp.busy
	prev := fp.prev

	switch {
	case busy != 0 && (busy < 0 || !statEqual(&prev, &curr)):
		fp.prev = curr
		fp.busy = 1
		fp.mu.Unlock()
		fp.cb(fp.owner, 0, &prev, &curr)
	case busy == 0:
		// First successful stat: record baseline, no callback (libuv fs-poll).
		fp.prev = curr
		fp.busy = 1
		fp.mu.Unlock()
	default:
		fp.prev = curr
		fp.busy = 1
		fp.mu.Unlock()
	}

	d := fp.interval - (time.Since(j.start) % fp.interval)
	j.resume <- d
}
