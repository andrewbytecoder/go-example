package main

import "fmt"

type DB struct {
	id int
}

func NewDB(id int) *DB {
	return &DB{id: id}
}

func (db DB) Query(string2 string) {
	fmt.Printf("使用 %d 连接对象 进行查询操作 %s\n", db.id, string2)
}

type DBPool struct {
	pool   map[int]*DB
	nextID int
}

func NewDBPool(num int) *DBPool {
	var pool = map[int]*DB{}
	for i := 0; i < num; i++ {
		pool[i] = NewDB(i)
	}
	return &DBPool{
		pool:   pool,
		nextID: num - 1,
	}
}

// GetDB 从连接池里面获取对象
func (p *DBPool) GetDB() *DB {
	if len(p.pool) > 0 {
		for id, db := range p.pool {
			delete(p.pool, id)
			return db
		}
	}
	p.nextID++
	db := NewDB(p.nextID)
	return db
}

// Release 放回对象到连接池
func (p *DBPool) Release(db *DB) {
	p.pool[db.id] = db
}
func main() {
	// 享元模式：将对象的可共享部分和不可共享部分分开处理
	// 比如共享单车和百度网盘
	// 最常用的就是内存预热和连接池的使用
	// 简单来说享元模式就是各种池
	pool := NewDBPool(1)
	db1 := pool.GetDB()
	db1.Query("select * from xxx")
	pool.Release(db1)
	db2 := pool.GetDB()
	db2.Query("select * from xxx")
	pool.Release(db2)
	db3 := pool.GetDB()
	db3.Query("select * from xxx")
	pool.Release(db3)
}
