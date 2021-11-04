package goflow

import (
	"context"
	"errors"
	"testing"
	"time"
)

type ReqInfo struct {
	srcID string
}

func TestNew(t *testing.T) {
	gf := New()
	if gf == nil {
		t.Error("New() error")
	}
}

func TestAdd1(t *testing.T) {
	ctx := context.Background()

	gf := New()
	gf.Add("test", false, []string{"dep1"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "test result", 0, nil
	})
	_, err := gf.Do(ctx)

	if err.Error() != "Error: Function \"dep1\" not exists!" {
		t.Error("Not existing function error")
	}
}

func TestAdd2(t *testing.T) {
	gf := New()
	ctx := context.Background()
	gf.Add("test", false, []string{"test"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "test result", 0, nil
	})
	_, err := gf.Do(ctx)

	if err.Error() != "Error: Function \"test\" depends of it self!" {
		t.Error("Self denepdency error")
	}
}

func TestDo1(t *testing.T) {
	ctx := context.Background()
	gf := New()

	gf.Add("test", false, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "test result", 0, nil
	})
	res, err := gf.Do(ctx)

	if err != nil || res["test"] != "test result" {
		t.Error("Incorrect result")
	}
}

func TestDo2(t *testing.T) {
	var shouldBeFalse bool = false
	ctx := context.Background()

	gf := New()
	gf.Add("first", false, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		time.Sleep(time.Second * 1)
		shouldBeFalse = true
		return "first result", 0, nil
	})
	gf.Add("second", false, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		shouldBeFalse = false
		return "second result", 0, nil
	})
	_, err := gf.Do(ctx)

	if err != nil || shouldBeFalse == true {
		t.Error("Incorrect goroutines execution order")
	}
}

func TestDo3(t *testing.T) {
	ctx := context.Background()

	gf := New()
	gf.Add("first", false, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "first result", 0, nil
	})
	gf.Add("second", false, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "second result", 0, nil
	})
	res, err := gf.Do(ctx)

	firstResult := res["first"]
	secondResult := res["second"]

	if err != nil || firstResult != "first result" || secondResult != "second result" {
		t.Error("Incorrect results")
	}
}

func TestDo4(t *testing.T) {
	ctx := context.Background()

	gf := New()
	gf.Add("first", false, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "first result", 0, errors.New("some error")
	})
	gf.Add("second", false, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "second result", 0, nil
	})
	_, err := gf.Do(ctx)

	if err.Error() != "execute err, please check task" {
		t.Error("Incorrect error value")
	}
}

func TestAwaysPanic(t *testing.T) {
	ctx := context.Background()

	gf := New()
	gf.Add("first", true, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		panic("test")
		return "first result", 0, errors.New("some error")
	})
	gf.Add("second", true, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		panic("test")
		return "second result", 0, nil
	})
	gf.Add("third", true, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		panic("test")
		return "third result", 0, nil
	})
	_, err := gf.Do(ctx)
	if err != nil {
		t.Error("Incorrect error value")
	}
}

func TestDoSkipPanic(t *testing.T) {
	ctx := context.Background()

	gf := New()
	gf.Add("first", true, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		panic("test")
		return "first result", 0, errors.New("some error")
	})
	gf.Add("second", false, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "second result", 0, nil
	})
	res, err := gf.Do(ctx)
	firstResult := res["first"]
	secondResult := res["second"]
	if err != nil || firstResult != nil || secondResult != "second result" {
		t.Error("Incorrect results")
	}
	if gf.Funcs["first"].ErrCode != ErrCodePanic || gf.Funcs["second"].ErrCode == ErrCodePanic {
		t.Error("Incorrect ErrCode")
	}
}


func TestDoSkipErr(t *testing.T) {
	ctx := context.Background()

	const UserDefineErrCode = 7
	gf := New()
	gf.Add("first", true, []string{}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "first result", UserDefineErrCode, errors.New("some error")
	})
	gf.Add("second", false, []string{"first"}, func(ctx context.Context, res map[string]interface{}) (
		interface{}, int64, error) {
		return "second result", 0, nil
	})
	res, err := gf.Do(ctx)
	firstResult := res["first"]
	secondResult := res["second"]
	if err != nil || firstResult != "first result" || secondResult != "second result" {
		t.Error("Incorrect results")
	}
	if gf.Funcs["first"].ErrCode != UserDefineErrCode || gf.Funcs["second"].ErrCode != 0 {
		t.Error("Incorrect ErrCode")
	}
}