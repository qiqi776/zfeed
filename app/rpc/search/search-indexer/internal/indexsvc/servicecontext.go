package indexsvc

import (
	"gorm.io/gorm"

	"zfeed/app/rpc/search/search-indexer/internal/indexconfig"
	"zfeed/app/rpc/search/search-indexer/internal/indexer"
	"zfeed/app/rpc/search/search-indexer/internal/indexrepo"
	"zfeed/orm"
)

type ServiceContext struct {
	Config     indexconfig.Config
	MysqlDb    *gorm.DB
	Repository *indexrepo.Repository
	Indexer    indexer.Indexer
}

func NewServiceContext(c indexconfig.Config) *ServiceContext {
	db := orm.MustNewMysql(&orm.Config{
		DSN:     c.MySQL.DataSource,
		Service: "search-indexer",
	})
	idx, err := indexer.New(c.IndexEngine)
	if err != nil {
		panic(err)
	}

	return &ServiceContext{
		Config:     c,
		MysqlDb:    db,
		Repository: indexrepo.New(db),
		Indexer:    idx,
	}
}
