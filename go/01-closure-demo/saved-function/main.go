package main

import "fmt"

func newCounter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

type CounterBox struct {
	next func() int // 函数值也可以保存到结构体字段。
}

func main() {
	counter := newCounter()
	box := CounterBox{next: counter} // 只保存函数，没有调用。

	fmt.Println(counter())  // 1
	fmt.Println(box.next()) // 2：使用同一份 count。

	copied := box.next
	fmt.Println(copied()) // 3：复制函数值后，仍使用同一份 count。
}
