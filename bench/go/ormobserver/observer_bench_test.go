package ormobserver

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	zfeedorm "zfeed/orm"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type observerBenchModel struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"size:32"`
}

var observerBenchDBSeq atomic.Uint64

func init() {
	logx.SetWriter(logx.NewWriter(io.Discard))
}

func BenchmarkObserverPluginQuery(b *testing.B) {
	b.Run("plain", func(b *testing.B) {
		db := openObserverBenchDB(b, false)
		benchmarkObserverQuery(b, db)
	})

	b.Run("observer", func(b *testing.B) {
		db := openObserverBenchDB(b, true)
		benchmarkObserverQuery(b, db)
	})
}

func BenchmarkObserverPluginCreate(b *testing.B) {
	b.Run("plain", func(b *testing.B) {
		db := openObserverBenchDB(b, false)
		benchmarkObserverCreate(b, db)
	})

	b.Run("observer", func(b *testing.B) {
		db := openObserverBenchDB(b, true)
		benchmarkObserverCreate(b, db)
	})
}

func openObserverBenchDB(b *testing.B, observer bool) *gorm.DB {
	b.Helper()

	name := strings.NewReplacer("/", "_", " ", "_").Replace(b.Name())
	seq := observerBenchDBSeq.Add(1)
	dsn := fmt.Sprintf("file:%s_%d?mode=memory&cache=shared", name, seq)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		b.Fatalf("open sqlite: %v", err)
	}

	if observer {
		if err := db.Use(zfeedorm.NewObserverPlugin("bench-orm", time.Hour)); err != nil {
			b.Fatalf("use observer plugin: %v", err)
		}
	}

	if err := db.AutoMigrate(&observerBenchModel{}); err != nil {
		b.Fatalf("auto migrate: %v", err)
	}
	if err := db.Create(&observerBenchModel{ID: 1, Name: "seed"}).Error; err != nil {
		b.Fatalf("seed model: %v", err)
	}

	return db
}

func benchmarkObserverQuery(b *testing.B, db *gorm.DB) {
	var loaded observerBenchModel
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		loaded = observerBenchModel{}
		if err := db.Where("id = ?", 1).Take(&loaded).Error; err != nil {
			b.Fatalf("query model: %v", err)
		}
	}
}

func benchmarkObserverCreate(b *testing.B, db *gorm.DB) {
	var id int64 = 1000
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		id++
		if err := db.Create(&observerBenchModel{ID: id, Name: "bench"}).Error; err != nil {
			b.Fatalf("create model: %v", err)
		}
	}
}
