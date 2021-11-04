/*
MIT License

Copyright (c) 2016 kamildrazkiewicz

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package goflow

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"time"
	"sync/atomic"
)

// Flow process data
type Flow struct {
	Funcs map[string]*flowStruct
}

const ErrCodePanic = 102420484096 // panic error code
const NormalReturnCode = 0        // normal return code

type flowFunc func(ctx context.Context, res map[string]interface{}) (interface{}, int64, error)

type flowStruct struct {
	//task deps data
	Deps []string
	//task deps number
	Ctr int
	//callback
	Fn flowFunc

	Name string
	//task start time
	StartAt time.Time
	//task duration
	Duration int64

	//failthrough
	FailThrough bool
	//errorCode: 102420484096 as panic magic number
	ErrCode int64
	//panic info
	Err error

	//transfer data
	C    chan interface{}
	once sync.Once
}

func (fs *flowStruct) Done(r interface{}) {
	for i := 0; i < fs.Ctr; i++ {
		fs.C <- r
	}
}

func (fs *flowStruct) Close() {
	fs.once.Do(func() {
		close(fs.C)
	})
}

func (fs *flowStruct) Init() {
	fs.C = make(chan interface{}, fs.Ctr)
}

// New flow struct
func New() *Flow {
	return &Flow{
		Funcs: make(map[string]*flowStruct),
	}
}

func (flw *Flow) Add(name string, failThrough bool, d []string, fn flowFunc) *Flow {
	flw.Funcs[name] = &flowStruct{
		Name:    name,
		Deps:    d,
		Fn:      fn,
		FailThrough: failThrough,
		Ctr:     1, // prevent deadlock
	}
	return flw
}

func (flw *Flow) Do(ctx context.Context) (map[string]interface{}, error) {
	for name, fn := range flw.Funcs {
		for _, dep := range fn.Deps {
			// prevent self depends
			if dep == name {
				return nil, fmt.Errorf("Error: Function \"%s\" depends of it self!", name)
			}
			// prevent no existing dependencies
			if _, exists := flw.Funcs[dep]; exists == false {
				return nil, fmt.Errorf("Error: Function \"%s\" not exists!", dep)
			}
			flw.Funcs[dep].Ctr++
		}
	}
	return flw.do(ctx)
}

func (flw *Flow) do(ctx context.Context) (map[string]interface{}, error) {
	// not a good idea, set err when lots of task failed, potential race condition
	var err error
	var ret atomic.Value
	ret.Store(int(0))
	res := make(map[string]interface{}, len(flw.Funcs))

	for _, f := range flw.Funcs {
		f.Init()
	}
	for name, f := range flw.Funcs {
		go func(name string, fs *flowStruct) {
			defer func() {
				// recover on panic
				if e := recover(); e != nil {
					var buf [4096]byte
					n := runtime.Stack(buf[:], false)
					fs.Err = errors.New(fs.Name + ":Panic:" + string(buf[:n]))
					fmt.Println(fs.Err.Error())
					fs.ErrCode = ErrCodePanic

					// failthrough off，transfer err，stop flow task
					if !fs.FailThrough {
						ret.Store(int(1))
					}
				}
				// safe close
				fs.Close()
				fs.Duration = time.Since(fs.StartAt).Milliseconds()
			}()
			results := make(map[string]interface{}, len(fs.Deps))
			// wating depends
			for _, dep := range fs.Deps {
				results[dep] = <-flw.Funcs[dep].C
			}
			fs.StartAt = time.Now()
			// check failthrough
			if  ret.Load().(int) > 0 && !fs.FailThrough {
				return
			}
			r, fnErrCode, fnErr := fs.Fn(ctx, results)
			fs.ErrCode = fnErrCode
			fs.Err = fnErr
			// failthrough on
			if fs.FailThrough {
				fs.Done(r)
				return
			}
			// failthrough off
			if fs.Err != nil || fs.ErrCode != NormalReturnCode {
				ret.Store(int(1))
			}
			// write result
			fs.Done(r)

		}(name, f)
	}

	// wait for all
	for name, fs := range flw.Funcs {
		res[name] = <-fs.C
	}

	if ret.Load().(int) > 0 {
		err = errors.New("execute err, please check task")
	}
	return res, err
}
