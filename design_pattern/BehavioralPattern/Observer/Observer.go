package main

import "fmt"

type Observer interface {
	RevMsg(wd int)
}

type User struct {
	Name string
}

func (u User) RevMsg(wd int) {
	fmt.Printf("%s 现在温度%d\n", u.Name, wd)
}

type Subject interface {
	SendMsg(wd int)
	Notify()
	RegisterObserver(Observer)
}

type WeatherZhan struct {
	observerList []Observer
	wd           int
}

func (w *WeatherZhan) SendMsg(wd int) {
	w.wd = wd
	w.Notify()
}
func (w *WeatherZhan) Notify() {
	for _, observer := range w.observerList {
		observer.RevMsg(w.wd)
	}
}
func (w *WeatherZhan) RegisterObserver(observer Observer) {
	w.observerList = append(w.observerList, observer)
}

func main() {
	// 当一个被被观察者的对象状态发生变化时，所有观察者都能收到变化同志
	zhan := WeatherZhan{}
	u1 := User{Name: "枫枫"}
	u2 := User{Name: "张三"}
	zhan.RegisterObserver(u1)
	zhan.RegisterObserver(u2)

	zhan.SendMsg(8)
	zhan.SendMsg(0)
}
