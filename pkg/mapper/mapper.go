package mapper

import (
	"strings"
	"ws-quant/pkg/feishu"
	"ws-quant/pkg/gintool"
	logger "ws-quant/pkg/log"
	"xorm.io/xorm"
)

var log = logger.NewLog("mapper")

func GetByWhere(engine *xorm.Engine, resultModel interface{}, where string, args ...interface{}) (has bool) {
	has, err := engine.Where(where, args...).Get(resultModel)
	if err != nil {
		log.Info("获取数据异常: ", err)
		feishu.Send("db查询异常," + err.Error())
	}
	return has
}

// Get 不支持默认字段的查询，如 false 和 空字符串
func Get(engine *xorm.Engine, model interface{}) (has bool) {
	has, err := engine.Get(model)
	if err != nil {
		log.Info("获取数据异常: ", err)
		feishu.Send("db查询异常," + err.Error())
	}
	return has
}

func FindLast(s *xorm.Engine, list interface{}, condBean interface{}) (err error) {
	err = s.Limit(1).Desc("id").Find(list, condBean)
	if err != nil {
		log.Error("数据库操作失败", "具体原因", err)
	}
	return err
}

func FindPage(s *xorm.Engine, list interface{}, where string, pager *gintool.Pager, args ...interface{}) (err error) {
	session := s.NewSession()
	if len(strings.TrimSpace(where)) > 0 {
		session = session.Where(where, args...)
	}
	if pager != nil {
		session = session.Limit(pager.PageSize, pager.NumStart)
	}
	err = session.Desc("id").Find(list)
	if err != nil {
		log.Error("数据库操作失败", "具体原因", err)
	}
	return err
}

func Count(s *xorm.Engine, model interface{}) (amount int64, err error) {
	amount, err = s.Count(model)
	return amount, err
}

// Insert 返回 code
func Insert(s *xorm.Engine, ele interface{}) (err error) {
	n, err := s.InsertOne(ele)
	if err != nil {
		log.Info("数据库插入链错误：", "err", err.Error())
		feishu.Send("db插入异常," + err.Error())
		return err
	}
	if n < 1 {
		log.Info("插入数据影响行数： 0")
	}
	return err
}

func UpdateById(engine *xorm.Engine, id interface{}, ele interface{}) int64 {
	num, err := engine.ID(id).Update(ele)
	if err != nil {
		log.Error("update fail:%v ", err)
		feishu.Send("update 数据失败:" + err.Error())
	}
	return num
}

func UpdateByWhere(engine *xorm.Engine, ele interface{}, whereClause string, args ...interface{}) int64 {
	num, err := engine.Where(whereClause, args...).Update(ele)
	if err != nil {
		log.Error("update fail:%v ", err)
		feishu.Send("update 数据失败:" + err.Error())
	}
	return num
}

func DeleteById(engine *xorm.Engine, id interface{}, ele interface{}) (num int64, err error) {
	return engine.ID(id).Delete(ele)
}

func Exec(engine *xorm.Engine, sql string) (err error) {
	_, err = engine.Exec(sql)
	if err != nil {
		log.Error("表更新数据失败：", "err", err)
	}
	return err
}
