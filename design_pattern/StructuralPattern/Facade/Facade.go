package main

import "fmt"

type Inventory struct {
}

func (k Inventory) Deduction() {
	fmt.Println("扣库存")
}

type Pay struct {
}

func (k Pay) Pay() {
	fmt.Println("支付")
}

type Logistics struct {
}

func (k Logistics) SendOutGoods() {
	fmt.Println("发货")
}

type Order struct {
	inventory *Inventory
	pay       *Pay
	logistics *Logistics
}

func NewOrder() *Order {
	return &Order{
		inventory: &Inventory{},
		pay:       &Pay{},
		logistics: &Logistics{},
	}
}

func (o Order) Place() {
	o.inventory.Deduction()
	o.pay.Pay()
	o.logistics.SendOutGoods()
}

func main() {
	// 外观模式就是用来简化接口，
	// 将原先复杂的功能进行集合，然后提供一个简化的接口供外部使用
	// 用来隐藏后端系统的复杂功能，后端封装一切事情就是在进行外观模式
	// 简单来说封装接口就是在进行外观模式的封装
	o := NewOrder()
	o.Place()
}
