package indexconfig

import "github.com/zeromicro/go-queue/kq"

type Config struct {
	Name           string
	MySQL          MySQLConf
	KqConsumerConf kq.KqConf
	IndexEngine    IndexEngineConf
}

type MySQLConf struct {
	DataSource string
}

type IndexEngineConf struct {
	Type           string
	Endpoint       string
	ContentIndex   string
	UserIndex      string
	Username       string
	Password       string
	TimeoutMs      int
	CompareEnabled bool
}
