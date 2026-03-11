package json

import "encoding/json"

type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type People struct {
	Data interface{} `json:"Person"`
}

func Unmarshal() error {
	person := Person{
		Name: "zhangsan",
		Age:  18,
	}

	people := People{
		Data: person,
	}

	data, err := json.Marshal(&people)
	if err != nil {
		return err
	}

	println(string(data))

	var people2 People
	err = json.Unmarshal(data, &people2)
	if err != nil {
		return err
	}
	// 这里会报错，当 Data 为interface{} 时，默认解析出来的类型为 map[string]interface{} 而不是对应的结构体
	person2 := people2.Data.(Person)
	println(person2.Name, person2.Age)

	return nil
}
