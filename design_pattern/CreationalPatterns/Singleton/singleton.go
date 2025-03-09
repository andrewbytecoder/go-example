package main

import (
	"fmt"
	"sync"
)

type DB struct {
}

var db *DB

func InitDB(dsn string) *DB {
	return &DB{}
}

var once sync.Once

func GetDB() *DB {
	once.Do(func() {
		db = InitDB("")
	})
	return db
}

func main() {
	d1 := GetDB()
	d2 := GetDB()
	d3 := GetDB()

	fmt.Printf("%p\n", d1)
	fmt.Printf("%p\n", d2)
	fmt.Printf("%p\n", d3)
}
