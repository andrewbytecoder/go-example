package main

import "fmt"

// 支付产品
type Pay interface {
	PayPage(price int64) string
}

// 微信支付，支付宝支付
type AliPay struct {
}

func (AliPay) PayPage(price int64) string {
	return "支付宝支付"
}

type WeiXinPay struct {
}

func (WeiXinPay) PayPage(price int64) string {
	return "微信支付"
}

// 退款产品
type Refund interface {
	Refund(no string) error
}

type AliRefund struct {
}

func (AliRefund) Refund(no string) error {
	fmt.Println("支付宝退款啦")
	return nil
}

type WeiXinRefund struct {
}

func (WeiXinRefund) Refund(no string) error {
	fmt.Println("微信退款啦")
	return nil
}

type PayFactory interface {
	CreatePay() Pay
	CreateRefund() Refund
}

type AliPayFactory struct {
}

func (AliPayFactory) CreatePay() Pay {
	// 专门管支付宝支付的相关逻辑
	return AliPay{}
}
func (AliPayFactory) CreateRefund() Refund {
	return AliRefund{}
}

type WeiXinPayFactory struct {
}

func (WeiXinPayFactory) CreatePay() Pay {
	return WeiXinPay{}
}
func (WeiXinPayFactory) CreateRefund() Refund {
	return WeiXinRefund{}
}

func main() {
	aliPayFactory := AliPayFactory{}
	aliPay := aliPayFactory.CreatePay()
	fmt.Println(aliPay.PayPage(15))
	aliPayFactory.CreateRefund().Refund("")

	weiXinPayFactory := WeiXinPayFactory{}
	weiXinPay := weiXinPayFactory.CreatePay()
	fmt.Println(weiXinPay.PayPage(15))
	weiXinPayFactory.CreateRefund().Refund("")
}
