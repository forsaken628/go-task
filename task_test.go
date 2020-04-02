package task

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

type Sum struct {
	args   []int
	result int
}

func (s *Sum) Inject(SubTaskAdder) func(context.Context) error {
	return func(context.Context) error {
		for _, i := range s.args {
			s.result += i
		}
		return nil
	}
}

func (s *Sum) Task() *Task {
	return NewTask(s)
}

type MulSum struct {
	args      [][]int
	subResult []*int
	result    []int
	async     bool
}

func (ms *MulSum) Inject(adder SubTaskAdder) func(context.Context) error {
	for _, v := range ms.args {
		s := &Sum{
			args: v,
		}
		ms.subResult = append(ms.subResult, &s.result)
		adder(s.Task(), ms.async)
	}
	return func(context.Context) error {
		ms.result = make([]int, 0)
		for _, v := range ms.subResult {
			ms.result = append(ms.result, *v)
		}
		return nil
	}
}

func (ms *MulSum) Task() *Task {
	return NewTask(ms)
}

func TestTask(t *testing.T) {
	t.Run("Sum", func(t *testing.T) {
		sum := &Sum{
			args:   []int{1, 2, 3, 5},
			result: 0,
		}

		err := sum.Task().Run(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 11, sum.result)
	})

	t.Run("MulSum", func(t *testing.T) {
		sum := &MulSum{
			args: [][]int{
				{1, 2, 3, 5},
				{1, 2},
				{1, 5},
			},
			async: false,
		}

		err := sum.Task().Run(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, []int{11, 3, 6}, sum.result)
	})

	t.Run("MulSum async", func(t *testing.T) {
		sum := &MulSum{
			args: [][]int{
				{1, 2, 3, 5},
				{1, 2},
				{1, 5},
			},
			async: true,
		}

		err := sum.Task().Run(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, []int{11, 3, 6}, sum.result)
	})

	t.Run("MulSum mix", func(t *testing.T) {

		sum0 := &Sum{
			args:   []int{1, 2},
			result: 0,
		}
		tk0 := sum0.Task()

		sum1 := &Sum{
			args:   []int{2, 2},
			result: 0,
		}
		tk1 := sum1.Task()

		sum2 := &Sum{
			args:   []int{4, 2},
			result: 0,
		}
		tk2 := sum2.Task()

		tk := NewTask(FuncInjectMaker(func(adder SubTaskAdder) func(context.Context) error {
			adder(tk0, true)
			adder(tk1, true)
			adder(tk2, false)
			return func(context.Context) error {
				return nil
			}
		}))

		err := tk.Run(context.Background())
		assert.Nil(t, err)
		assert.Equal(t, 3, sum0.result)
		assert.Equal(t, 4, sum1.result)
		assert.Equal(t, 6, sum2.result)
	})
}

func main() {

	sum0 := &Sum{
		args:   []int{1, 2},
		result: 0,
	}
	tk0 := sum0.Task()

	sum1 := &Sum{
		args:   []int{2, 2},
		result: 0,
	}
	tk1 := sum1.Task()

	sum2 := &Sum{
		args:   []int{4, 2},
		result: 0,
	}
	tk2 := sum2.Task()

	tk := NewTask(FuncInjectMaker(func(adder SubTaskAdder) func(context.Context) error {
		adder(tk0, true)
		adder(tk1, true)
		adder(tk2, false)
		return func(context.Context) error {
			return nil
		}
	}))

	err := tk.Run(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Println(sum0.result) // 3
	fmt.Println(sum1.result) // 4
	fmt.Println(sum2.result) // 6
}
