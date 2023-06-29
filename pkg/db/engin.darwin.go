package db

import (
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"xorm.io/xorm"
)

type Config struct {
	DriverName string `yaml:"driverName"`
	Ip         string `yaml:"ip"`
	Port       int    `yaml:"port"`
	Usr        string `yaml:"usr"`
	Pwd        string `yaml:"pwd"`
	Schema     string `yaml:"schema"`
}

func New(cnf *Config) *xorm.Engine {
	// 注意下面的这种后缀 ?会报错
	//format := "%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true&loc=Asia/Shanghai"
	format := "%s:%s@tcp(%s:%d)/%s"
	dsn := fmt.Sprintf(format, cnf.Usr, cnf.Pwd, cnf.Ip, cnf.Port, cnf.Schema)
	log.Printf("dsn: %v\n", dsn)
	engine, err := xorm.NewEngine(cnf.DriverName, dsn)
	if err != nil {
		log.Panic("create xml engine error ", err)
	}
	engine.ShowSQL(false)
	err = engine.Ping()
	if err != nil {
		log.Panic("连接数据库异常 ", err)
	}
	log.Println("成功连接数据库:", dsn)
	return engine
}
