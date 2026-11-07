# Go 闭包：从计数器到保存函数

**闭包就是：函数保留了访问外层变量的能力，即使外层函数已经返回。**

先看 [main.go](main.go) 中的最小计数器，再看 [saved-function/main.go](saved-function/main.go) 中把闭包保存到结构体字段的例子。两个例子都只使用标准库，不需要额外服务。

运行计数器：

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

## 1. 先看最小的闭包

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

可以这样理解：

```text
counter：函数值
  ├── 要执行的代码：count++; return count
  └── 可以访问的外层变量：那一次 newCounter 调用创建的 count
```

这是一张帮助理解的概念图，不表示编译器必须使用某一种内存布局。

## 2. 一步步看执行过程

1. `counter := newCounter()`：创建 `count = 0`，把里面的函数返回并保存到 `counter`。此时还没有执行 `count++`。
2. 第一次 `counter()`：执行返回的函数，把那份 `count` 加到 1。
3. 第二次 `counter()`：继续使用同一份 `count`，加到 2。
4. 第三次 `counter()`：继续加到 3。
5. `other := newCounter()`：重新调用外层函数，创建另一份独立的 `count = 0`。
6. `other()` 返回 1，之后 `counter()` 返回 4，两者各自保存自己的状态。

`newCounter()` 是创建计数器，`counter()` 是使用已经创建的计数器。两者调用的是不同的函数。

外层函数返回后，外部不能直接使用局部名字 `count`，但返回的函数仍然可以访问它。Go 会保证仍需访问的数据有效，不会因为外层函数返回就立即销毁它。

这里需要分清：**名字的作用域结束，与数据是否仍需要被访问，是两件事。**

## 3. 函数本身也能当作值保存

先区分两个动作：

```go
counter := newCounter() // 执行外层函数，获得里面的函数值。
saved := counter       // 保存这个函数值，没有执行 count++。
fmt.Println(saved())   // 调用保存的函数，此时才执行 count++。
```

函数值可以赋给变量、作为参数传递，也可以保存到结构体字段：

```go
type CounterBox struct {
    next func() int
}

counter := newCounter()
box := CounterBox{next: counter}

fmt.Println(counter()) // 1
fmt.Println(box.next()) // 2

copied := box.next
fmt.Println(copied()) // 3
```

`next func() int` 表示 `next` 字段保存一个“无参数、返回 int”的函数。

`counter`、`box.next`、`copied` 都来自同一个闭包函数值，因此访问同一份 `count`。复制函数值不会重新执行 `newCounter()`，也不会创建一份独立的计数。

相比之下，`other := newCounter()` 会再次执行外层函数，创建一份新的 `count`。这就是“保存已有函数”和“重新创建闭包”的区别。

上面这段代码可以直接运行：

```bash
go run ./saved-function
```

输出为 `1`、`2`、`3`，每个数字各占一行。

持有关系是：

```text
box
  └── next 字段：函数值
        └── 能继续访问原来的 count
```

## 4. 捕获的是变量，不是创建时的旧值

```go
n := 10
show := func() { fmt.Println(n) }
n = 20
show() // 20
```

`show` 访问的是同一个 `n` 变量，因此执行时看到修改后的 20。

如果希望保留当时的数字，可以先存进另一个变量：

```go
n := 10
saved := n
show := func() { fmt.Println(saved) }
n = 20
show() // 10
```

这个闭包访问的是 `saved`，它没有被修改。如果后来再给 `saved` 赋值，回调同样会看到新值。

如果变量保存的是指针，`saved := original` 复制的是指针值，两个指针仍可能指向同一个对象；不会自动深拷贝那个对象。

## 5. 为什么外层函数返回后还能访问？

Go 会根据数据的生命周期安排存储。需要在函数返回后继续使用的数据，常见实现方式包括逃逸到堆；这不意味着整个外层函数一直不返回，也不意味着其完整栈帧一直被保留。

只要程序后续仍能通过闭包访问 `count`，Go 就必须保证访问有效。

当所有需要这份状态的闭包引用都不再可达时，相关存储才可能被回收。例如，只清空 `box.next`，但仍保存着 `counter` 或 `copied`，就没有移除对这份状态的全部访问入口。满足回收条件也不代表立即发生垃圾回收。

## 6. 匿名函数、闭包、回调分别在说什么？

| 概念 | 关注什么 | 例子 |
| --- | --- | --- |
| 匿名函数 | 函数没有声明名字 | `func() { fmt.Println("hello") }` |
| 闭包 | 函数可以访问捕获的外层变量 | `func() { count++ }` 中的 `count` 来自外层 |
| 回调 | 把函数交给另一个组件，由那个组件在需要时调用 | 将函数交给订阅方法，等消息到来时执行 |

同一个函数可以同时具备这三个特征。闭包也不一定用于回调，最开始的计数器就是由调用方自己直接调用。

## 7. 再对应最初的回调场景

理解 `box.next = counter` 后，就容易理解 `consumer.handler = callback`：两者都是把函数值保存到结构体字段。

计数器函数使用外层的 `count`；最初的消息回调使用外层的 `dispatcher`。对应的关系是：

```text
消费者
  └── handler 字段：保存的回调函数值
        └── 能访问外层的 dispatcher 变量
              └── 变量中的指针指向 Dispatcher 对象
                    └── handlers map
                          └── 业务 Handler
```

保存回调时还没有执行函数体，收到消息并调用回调时才会执行。回调通过捕获的变量继续访问对象；这里没有把 Dispatcher 和所有业务 Handler 自动复制一遍。

**闭包让函数可以带着状态继续执行；把这个函数保存起来，也就保留了通过它访问这些状态的入口。**
