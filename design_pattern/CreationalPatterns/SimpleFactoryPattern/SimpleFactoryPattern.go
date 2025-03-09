package main

import "fmt"

// Pay 支付产品
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

type PayType int8

const (
	AliPayType    PayType = 1
	WeiXinPayType PayType = 2
)

func NewPayPage(payType PayType) Pay {
	switch payType {
	case AliPayType:
		return AliPay{}
	case WeiXinPayType:
		return WeiXinPay{}
	}
	return nil
}

func main() {
	pay := NewPayPage(AliPayType)
	fmt.Println(pay.PayPage(12))
	pay = NewPayPage(WeiXinPayType)
	fmt.Println(pay.PayPage(12))
}
