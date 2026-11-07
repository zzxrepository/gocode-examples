package main

import "fmt"

// newCounter 返回一个函数。返回的函数仍能访问外层的 count 变量。
func newCounter() func() int {
	count := 0

	return func() int {
		count++
		return count
	}
}

func main() {
	counter := newCounter()
	fmt.Println(counter()) // 1
	fmt.Println(counter()) // 2
	fmt.Println(counter()) // 3

	other := newCounter()  // 再调用一次外层函数，创建另一份 count。
	fmt.Println(other())   // 1
	fmt.Println(counter()) // 4：原来的 count 不受影响。
}
