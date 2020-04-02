# TASK

* 这是一个抽象的任务调度模型，能在类型安全的前提下，把父子函数的调用转化为父子任务的调用。
* 一个父任务可以有多个子任务，子任务之间可以是并行执行，也可以是串行执行，由父控制，而且非常容易切换。
* 错误处理模式为一个失败，全部放弃。
* 其实有点像js的Promise，将执行逻辑的构造与实际执行过程拆分，后续考虑添加final，可以比较方便的完成资源回收一类的工作

```go
// 一个简单的加法任务
type Sum struct {
    // 入参
	args   []int
    // 出参
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

func main() {
	sum := &Sum{
		args:   []int{1, 2, 3, 5},
		result: 0,
	}

	err := sum.Task().Run(context.Background())
	if err != nil {
		panic(err)
	}
	
	fmt.Println(sum.result) // 11
}
```

```go
// 多个加法任务
type MulSum struct {
	args      [][]int
	subResult []*int
	result    []int
	async     bool
}

func (ms *MulSum) Inject(adder SubTaskAdder) func(context.Context) error {
    // 子任务执行之前，构造子任务
	for _, v := range ms.args {
		s := &Sum{
			args: v,
		}
		ms.subResult = append(ms.subResult, &s.result)
        // 非常容易转为异步执行
		adder(s.Task(), ms.async)
	}
	return func(context.Context) error {
        // 子任务执行之后，执行当前任务
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

func main() {
	sum := &MulSum{
		args: [][]int{
			{1, 2, 3, 5},
			{1, 2},
			{1, 5},
		},
		async: false,
	}

	err := sum.Task().Run(context.Background())
	if err != nil {
		panic(err)
	}
	
	fmt.Println(sum.result) //[]int{11, 3, 6}
}
```

```go
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

    //利用闭包函数，可以不写结构体构造任务，非常灵活
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
```
