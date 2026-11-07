# Go 闭包：一个计数器就能讲清楚

**闭包就是：函数保留了访问外层变量的能力，即使外层函数已经返回。**

完整代码见 [main.go](main.go)。运行：

```bash
cd /Users/mmzhang/notes/GoTutorials/gocode-examples/go/01-closure-demo
go run .
```

输出：

```text
1
2
3
1
4
```

## 核心代码

```go
func newCounter() func() int {
    count := 0

    return func() int {
        count++
        return count
    }
}
```

`newCounter() func() int` 的意思是：`newCounter` 不接收参数，返回一个“无参数、返回 int”的函数。

返回的匿名函数使用了外层声明的 `count`。这个函数与它能够访问的外层变量环境，一起构成闭包。

## 一步步看执行过程

1. `counter := newCounter()`：创建 `count = 0`，把里面的函数返回并保存到 `counter`。此时还没有执行 `count++`。
2. 第一次 `counter()`：执行返回的函数，把那份 `count` 加到 1。
3. 第二次 `counter()`：继续使用同一份 `count`，加到 2。
4. 第三次 `counter()`：继续加到 3。
5. `other := newCounter()`：重新调用外层函数，创建另一份独立的 `count = 0`。
6. `other()` 返回 1，之后 `counter()` 返回 4，两者各自保存自己的状态。

`newCounter()` 是创建计数器，`counter()` 是使用已经创建的计数器。两者调用的是不同的函数。

外层函数返回后，外部不能直接使用局部名字 `count`，但返回的函数仍然可以访问它。Go 会保证仍需访问的数据有效，不会因为外层函数返回就立即销毁它。

## 捕获的是变量，不是旧值

```go
n := 10
show := func() { fmt.Println(n) }
n = 20
show() // 20
```

`show` 访问的是同一个 `n` 变量，因此执行时看到修改后的 20。

记住：**闭包让函数可以带着状态继续执行。** 函数没有名字叫匿名函数；这里的匿名函数因为使用了外层变量，体现了闭包的作用。
