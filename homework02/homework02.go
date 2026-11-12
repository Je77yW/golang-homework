package main

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

func intPointer(num *int) {
	*num += 10
}

func slicePointer(nums *[]int) {
	for i := range *nums {
		(*nums)[i] *= 2
	}
}

func goRoutine() {
	var wg sync.WaitGroup
	for i := range 2 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 1; j <= 9; j += 2 {
				fmt.Printf("id %d print %d\n", id, id+j)
			}
		}(i)
	}
	wg.Wait()
}

func taskScheduler(tasks []func()) {
	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(id int, t func()) {
			defer wg.Done()
			start := time.Now()
			t()
			fmt.Printf("Task %d executed in %v\n", id, time.Since(start))
		}(i+1, task)
	}
	wg.Wait()
}

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

func (r Rectangle) Perimeter() float64 {
	return 2 * (r.Width + r.Height)
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 {
	return math.Pi * c.Radius * c.Radius
}

func (c Circle) Perimeter() float64 {
	return 2 * math.Pi * c.Radius
}

func printShapeInfo(s Shape) {
	fmt.Printf("Area: %.2f, Perimeter: %.2f\n", s.Area(), s.Perimeter())
}

type Person struct {
	Name string
	Age  uint
}

type Employee struct {
	P          Person
	EmployeeID int
}

func (e *Employee) printEmployeeInfo() {
	fmt.Printf("Name: %s, Age: %d, EmployeeID: %d\n", e.P.Name, e.P.Age, e.EmployeeID)
}

func channelWithoutBuffer() {
	ch := make(chan int)

	go func() {
		for i := range 10 {
			ch <- i + 1
		}
		close(ch)
	}()

	for value := range ch {
		fmt.Printf("received: %d\n", value)
	}
}

func channelWithBuffer() {
	ch := make(chan int, 3)
	var wg sync.WaitGroup

	go func() {
		for i := range 100 {
			ch <- i
		}
		close(ch)
	}()

	for i := range 3 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for value := range ch {
				fmt.Printf("Consumer %d received: %d\n", id, value)
			}
		}(i)
	}
	wg.Wait()
}

type SafeCounter struct {
	mu    sync.Mutex
	count int
}

func (sc *SafeCounter) increment() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.count++
}

func (sc *SafeCounter) getCount() int {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.count
}

func safeCount() {
	sc := SafeCounter{}
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				sc.increment()
			}
		}()
	}
	wg.Wait()
	fmt.Println(sc.getCount())
}

type AtomicCounter struct {
	count atomic.Int64
}

func (ac *AtomicCounter) increment() {
	ac.count.Add(1)
}

func (ac *AtomicCounter) getCount() int64 {
	return ac.count.Load()
}

func atomicCount() {
	var ac AtomicCounter
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				ac.increment()
			}
		}()
	}
	wg.Wait()
	fmt.Println(ac.getCount())
}

func main() {
	var a = 1
	intPointer(&a)
	fmt.Printf("IntPointer: %d\n", a)

	slice := []int{1, 2, 3}
	slicePointer(&slice)
	fmt.Printf("SlicePointer: ")
	fmt.Println(slice)

	goRoutine()

	fmt.Println("TaskScheduler:")
	taskScheduler([]func(){
		func() { time.Sleep(100 * time.Millisecond) },
		func() { time.Sleep(200 * time.Millisecond) },
		func() { time.Sleep(50 * time.Millisecond) },
	})

	rec := Rectangle{10, 5}
	cir := Circle{5}
	printShapeInfo(rec)
	printShapeInfo(cir)

	emp := Employee{Person{"Alice", 21}, 1}
	emp.printEmployeeInfo()

	channelWithoutBuffer()
	channelWithBuffer()

	safeCount()
	atomicCount()
}
