package template

/*
K 和 V 你可以理解为类型别名，在中括号之间进行定义，作用域也只在此函数内，可以在形参、函数主体、返回值类型 里使用
comparable 是 Go 语言预声明的类型，是那些可以比较（可哈希）的类型的集合，通常用于定义 map 里的 key 类型
int64 | float64 意思是 V 可以是 int64 或 float64 中的任意一个
map[K]V 就是使用了 K 和 V 这两个别名类型的 map

// SumIntsOrFloats 函数的泛型类型参数 K 和 V 的定义，K 是一个可哈希的类型，V 是一个 int64 或 float64 类型
*/
func SumIntsOrFloats[K comparable, V int64 | float64](m map[K]V) V {
	var s V
	for _, v := range m {
		s += v
	}
	return s
}
