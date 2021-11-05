# GoFlow
[![GoDoc](http://godoc.org/github.com/kamildrazkiewicz/go-flow?status.svg)](http://godoc.org/github.com/sinomiko/go-flow) [![License](https://img.shields.io/github/license/mashape/apistatus.svg?maxAge=2592000)](https://raw.githubusercontent.com/sinomiko/go-flow/master/LICENSE) [![Release](https://img.shields.io/github/release/kamildrazkiewicz/go-flow.svg?label=Release)](http://godoc.org/github.com/kamildrazkiewicz/go-flow) [![GoReport](https://goreportcard.com/badge/github.com/kamildrazkiewicz/go-flow)](https://goreportcard.com/report/github.com/kamildrazkiewicz/go-flow) [![Travis](https://travis-ci.org/sinomiko/go-flow.svg?branch=master)](https://travis-ci.org/sinomiko/go-flow) [![Coverage](http://coveralls.io/repos/github/sinomiko/go-flow/badge.svg?branch=master)](https://coveralls.io/github/sinomiko/go-flow?branch=master)

Safe DAG framework，GoFlow evolution version

Goflow is a simply package to control goroutines execution order based on dependencies. It works similar to ```async.auto``` from [node.js async package](https://github.com/caolan/async), but for Go.

![](http://i.imgur.com/Rej4XAC.png)

## realease LOG
### 20211104
- add task duration
- panic recover
- transfer error code, error msg
- FailThrough On: task panic, after task will continue
- solve race condition

## Install

Install the package with:

```bash
go get github.com/sinomiko/go-flow
```

Import it with:

```go
import "github.com/sinomiko/go-flow"
```

and use `goflow` as the package name inside the code.

## Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	goflow "github.com/sinomiko/go-flow"
)

// FlowData flow ctx携带信息
type FlowData struct {
	OuterReq         interface{}
	OuterRsp         interface{}
	AttachInfo       interface{}
	ReqType          uint64
	ProcessorID      string
}

// ReqInfo 请求信息
type ReqInfo struct {
	srcID string
}

func main() {
	doFlow1()
	doFlow2()
	doFlow3()
}

var f1 = func(ctx context.Context, r map[string]interface{}) (interface{}, int64, error) {
	fmt.Println("function1 started", r)
	flow_data, ok := ctx.Value("flowData").(*FlowData)
	if !ok || flow_data == nil {
		return nil, 0, fmt.Errorf("Error: flow_data is nil - %v", flow_data)
	}

	outer_req, ok := flow_data.OuterReq.(*ReqInfo)
	if !ok || outer_req == nil {
		return nil, 0, fmt.Errorf("Error: outer_req is nil - %v", outer_req)
	}
	fmt.Println("function1 end", "ctx:", ctx, "srcID:", outer_req.srcID)
	time.Sleep(time.Millisecond * 25)
	return "f1", 0, nil
}

var f2 = func(ctx context.Context, r map[string]interface{}) (interface{}, int64, error) {
	fmt.Println("function2 started", r["Start"])
	time.Sleep(time.Millisecond * 50)
	return "some results", 0, nil // errors.New("Some error")
}
var f3 = func(ctx context.Context, r map[string]interface{}) (interface{}, int64, error) {
	fmt.Println("function3 started", r["Start"])
	time.Sleep(time.Millisecond * 75)
	return nil, 0, nil
}

var f4 = func(ctx context.Context, r map[string]interface{}) (interface{}, int64, error) {
	fmt.Println("function4 started", r)
	time.Sleep(time.Millisecond * 100)
	return "f4 end", 0, nil
}

func doFlow1() {
	ctx := context.Background()
	flowData := &FlowData{
		OuterReq:    &ReqInfo{srcID: "1"},
		OuterRsp:    "",
		ReqType:     1,
		ProcessorID: "Flow1",
	}
	// 特殊数据 在ctx中从头透传到结尾
	valueCtx := context.WithValue(ctx, "flowData", flowData)

	// with valueCtx
	flow := goflow.New().
		Add("f1", false, []string{}, f1).
		Add("f2", false, []string{"f1"}, f2).
		Add("f3", false, []string{"f1"}, f3).
		Add("f4", false, []string{"f2", "f3"}, f4)
	res, err := flow.Do(valueCtx)

	fmt.Println("======flow1 f4: res======== ")
	fmt.Printf("res : %s\n", res["f4"])
	fmt.Println("======flow1 time stats======== ")
	for k, v := range flow.Funcs {
		fmt.Printf("%s -> duration[%d]\n", k, v.Duration)
	}
	fmt.Println("======flow1 result ======== ")
	fmt.Println(res, err)
}

func doFlow2(){
	fmt.Println("======new flow2======== ")
	ctx2 := context.Background()
	// with Ctx
	flow2 := goflow.New().
		Add("f1", false, []string{}, f1).
		Add("f2", false, []string{"f1"}, f2).
		Add("f3", false, []string{"f1"}, f3).
		Add("f4", false, []string{"f2", "f3"}, f4)
	res2, err2 := flow2.Do(ctx2)

	fmt.Println("======flow2 result ======== ")
	fmt.Println(res2, err2)
}

func doFlow3(){
	fmt.Println("======new flow3======== ")
	ctx3 := context.Background()
	fErr := func(ctx context.Context, r map[string]interface{}) (interface{}, int64, error) {
		fmt.Println("fErr started", r)
		time.Sleep(time.Millisecond * 100)
		panic("test")
		return "f4 end", 0, nil
	}
	// with panic func and skip panic
	flow3 := goflow.New().
		Add("fErr", true, []string{}, fErr).
		Add("f3", false, []string{"fErr"}, f3).
		Add("f4", false, []string{"fErr", "f3"}, f4)
	res3, err3 := flow3.Do(ctx3)

	fmt.Println("======flow3 result ======== ")
	fmt.Println(res3, err3)
}

```

Output will be:
```
function1 started map[]
function1 end ctx: context.Background.WithValue(type string, val <not Stringer>) srcID: 1
function2 started <nil>
function3 started <nil>
function4 started map[f2:some results f3:<nil>]
======flow1 f4: res======== 
res : f4 end
======flow1 time stats======== 
f3 -> duration[78]
f4 -> duration[105]
f1 -> duration[26]
f2 -> duration[53]
======flow1 result ======== 
...

```
