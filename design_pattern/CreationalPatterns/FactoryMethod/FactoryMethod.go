package main

import "fmt"

// 产品变动不需要修改其他代码符合开闭原则，简单工厂模式不符合
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

type YinLianPay struct {
}

func (YinLianPay) PayPage(price int64) string {
	return "银联支付"
}

type PayFactory interface {
	CreatePay() Pay
}

// 创建两个厂
type AliPayFactory struct {
}

func (AliPayFactory) CreatePay() Pay {
	// 专门管支付宝支付的相关逻辑
	return AliPay{}
}

type WeiXinPayFactory struct {
}

func (WeiXinPayFactory) CreatePay() Pay {
	return WeiXinPay{}
}

type YinLianPayFactory struct {
}

func (YinLianPayFactory) CreatePay() Pay {
	return YinLianPay{}
}

func main() {
	aliPayFactory := AliPayFactory{}
	aliPay := aliPayFactory.CreatePay()
	fmt.Println(aliPay.PayPage(15))

	weiXinPayFactory := WeiXinPayFactory{}
	weiXinPay := weiXinPayFactory.CreatePay()
	fmt.Println(weiXinPay.PayPage(15))

	yinLianPayFactory := YinLianPayFactory{}
	yinLianPay := yinLianPayFactory.CreatePay()
	fmt.Println(yinLianPay.PayPage(15))
}
