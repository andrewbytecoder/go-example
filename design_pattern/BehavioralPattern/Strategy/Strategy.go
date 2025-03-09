package main

import "fmt"

type Pay interface {
	Pay(money int64)
}
type AliPay struct {
}

func (AliPay) Pay(money int64) {
	fmt.Println("使用支付宝支付", money)
}

type WeiPay struct {
}

func (WeiPay) Pay(money int64) {
	fmt.Println("使用微信支付", money)
}

type PayStrategy struct {
	pay Pay
}

func (p *PayStrategy) SetPay(pay Pay) {
	p.pay = pay
}
func (p *PayStrategy) Pay(money int64) {
	p.pay.Pay(money)
}

func main() {
	// 定义一系列算法，并且算法之间能够相互替换
	// 最核心的思想就是将算法的实现和算法的使用进行分离
	// 这样但客户端需要更换算法时只需要更改算法就行而不必修改客户端代码
	// 简单来说就是对各个不同的工具提供一个代理，最经典的实现就是VFS虚拟文件系统
	aliPay := &AliPay{}
	weiPay := &WeiPay{}

	payStrategy := &PayStrategy{}
	payStrategy.SetPay(aliPay)
	payStrategy.Pay(12)
	payStrategy.SetPay(weiPay)
	payStrategy.Pay(12)

}
