package orm

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	gzprom "github.com/zeromicro/go-zero/core/prometheus"
	gztracetest "github.com/zeromicro/go-zero/core/trace/tracetest"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type observerTestModel struct {
	ID   int64  `gorm:"primaryKey"`
	Name string `gorm:"size:32"`
}

func TestNewMysqlValidateConfig(t *testing.T) {
	t.Parallel()

	if _, err := NewMysql(nil); err == nil {
		t.Fatalf("expected nil config error")
	}

	if _, err := NewMysql(&Config{}); err == nil {
		t.Fatalf("expected empty dsn error")
	}
}

func TestNormalizeService(t *testing.T) {
	t.Parallel()

	if got := normalizeService("  content-rpc "); got != "content-rpc" {
		t.Fatalf("normalizeService() = %q, want %q", got, "content-rpc")
	}

	if got := normalizeService("   "); got != "unknown" {
		t.Fatalf("normalizeService() = %q, want %q", got, "unknown")
	}
}

func TestTableFromSQL(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"SELECT * FROM `zfeed_user` WHERE id = 1":                    "zfeed_user",
		"INSERT INTO `zfeed_follow` (`user_id`) VALUES (1)":          "zfeed_follow",
		"UPDATE `zfeed_content` SET `status` = 30 WHERE id = 9":      "zfeed_content",
		"DELETE FROM `zfeed_comment` WHERE `id` = 1":                 "zfeed_comment",
		"select * from zfeed_article where content_id = 100":         "zfeed_article",
		"insert into zfeed_mq_consume_dedup (consumer) values ('x')": "zfeed_mq_consume_dedup",
	}

	for stmt, want := range cases {
		stmt := stmt
		want := want
		t.Run(stmt, func(t *testing.T) {
			t.Parallel()
			if got := tableFromSQL(stmt); got != want {
				t.Fatalf("tableFromSQL(%q) = %q, want %q", stmt, got, want)
			}
		})
	}
}

func TestCompactSQL(t *testing.T) {
	t.Parallel()

	got := compactSQL(" SELECT   *   FROM   zfeed_user \n WHERE id = 1 ")
	if got != "SELECT * FROM zfeed_user WHERE id = 1" {
		t.Fatalf("compactSQL() = %q", got)
	}

	longSQL := strings.Repeat("x", 600)
	compacted := compactSQL(longSQL)
	if len(compacted) <= 512 {
		t.Fatalf("expected truncated sql, got len=%d", len(compacted))
	}
}

func TestExtractTableFallsBackToSQL(t *testing.T) {
	t.Parallel()

	stmt := &gorm.Statement{}
	stmt.SQL.WriteString("SELECT * FROM `zfeed_user` WHERE id = 1")
	db := &gorm.DB{Statement: stmt}

	if got := extractTable(db); got != "zfeed_user" {
		t.Fatalf("extractTable() = %q, want %q", got, "zfeed_user")
	}
}

func TestObserverPluginRecordsMetrics(t *testing.T) {
	gzprom.Enable()

	service := fmt.Sprintf("orm-test-%d", time.Now().UnixNano())
	table := "observer_test_models"

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", service)), &gorm.Config{
		Logger: &observerLogger{level: logger.Silent},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.Use(NewObserverPlugin(service, time.Hour)); err != nil {
		t.Fatalf("use observer plugin: %v", err)
	}

	if err := db.AutoMigrate(&observerTestModel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	createBefore := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "create",
		"table":     table,
		"result":    "ok",
	})
	queryOKBefore := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "query",
		"table":     table,
		"result":    "ok",
	})
	queryMissBefore := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "query",
		"table":     table,
		"result":    "not_found",
	})
	updateBefore := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "update",
		"table":     table,
		"result":    "ok",
	})
	deleteBefore := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "delete",
		"table":     table,
		"result":    "ok",
	})
	histBefore := metricHistogramCount(t, "zfeed_db_statement_duration_ms", map[string]string{
		"service":   service,
		"operation": "create",
		"table":     table,
	})

	model := &observerTestModel{Name: "alpha"}
	if err := db.Create(model).Error; err != nil {
		t.Fatalf("create: %v", err)
	}

	var loaded observerTestModel
	if err := db.Where("name = ?", "alpha").First(&loaded).Error; err != nil {
		t.Fatalf("query existing: %v", err)
	}

	var missing observerTestModel
	if err := db.Where("name = ?", "missing").First(&missing).Error; err != gorm.ErrRecordNotFound {
		t.Fatalf("query missing: %v", err)
	}

	if err := db.Model(&loaded).Update("name", "beta").Error; err != nil {
		t.Fatalf("update: %v", err)
	}

	if err := db.Delete(&loaded).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}

	createAfter := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "create",
		"table":     table,
		"result":    "ok",
	})
	queryOKAfter := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "query",
		"table":     table,
		"result":    "ok",
	})
	queryMissAfter := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "query",
		"table":     table,
		"result":    "not_found",
	})
	updateAfter := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "update",
		"table":     table,
		"result":    "ok",
	})
	deleteAfter := metricCounterValue(t, "zfeed_db_statement_total", map[string]string{
		"service":   service,
		"operation": "delete",
		"table":     table,
		"result":    "ok",
	})
	histAfter := metricHistogramCount(t, "zfeed_db_statement_duration_ms", map[string]string{
		"service":   service,
		"operation": "create",
		"table":     table,
	})

	if createAfter-createBefore != 1 {
		t.Fatalf("create delta = %v, want 1", createAfter-createBefore)
	}
	if queryOKAfter-queryOKBefore != 1 {
		t.Fatalf("query ok delta = %v, want 1", queryOKAfter-queryOKBefore)
	}
	if queryMissAfter-queryMissBefore != 1 {
		t.Fatalf("query miss delta = %v, want 1", queryMissAfter-queryMissBefore)
	}
	if updateAfter-updateBefore != 1 {
		t.Fatalf("update delta = %v, want 1", updateAfter-updateBefore)
	}
	if deleteAfter-deleteBefore != 1 {
		t.Fatalf("delete delta = %v, want 1", deleteAfter-deleteBefore)
	}
	if histAfter-histBefore != 1 {
		t.Fatalf("histogram create count delta = %d, want 1", histAfter-histBefore)
	}
}

func TestObserverPluginCreatesDBSpan(t *testing.T) {
	db := openObserverTestDB(t, fmt.Sprintf("orm-trace-%d", time.Now().UnixNano()))
	me := gztracetest.NewInMemoryExporter(t)

	ctx, parent := otel.Tracer("orm-test").Start(context.Background(), "parent")
	model := &observerTestModel{Name: "trace"}
	if err := db.WithContext(ctx).Create(model).Error; err != nil {
		t.Fatalf("create: %v", err)
	}
	parent.End()

	span := findSpanStubByName(t, me.GetSpans(), "db.create observer_test_models")
	if span.Parent.SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("db span parent = %s, want %s", span.Parent.SpanID(), parent.SpanContext().SpanID())
	}
	if span.SpanKind != oteltrace.SpanKindClient {
		t.Fatalf("db span kind = %v, want client", span.SpanKind)
	}
	if got := spanAttributeValue(span.Attributes, "db.system"); got != "mysql" {
		t.Fatalf("db.system = %q, want mysql", got)
	}
	if got := spanAttributeValue(span.Attributes, "db.operation"); got != "create" {
		t.Fatalf("db.operation = %q, want create", got)
	}
	if got := spanAttributeValue(span.Attributes, "db.table"); got != "observer_test_models" {
		t.Fatalf("db.table = %q, want observer_test_models", got)
	}
	if got := spanAttributeValue(span.Attributes, "zfeed.db.result"); got != "ok" {
		t.Fatalf("zfeed.db.result = %q, want ok", got)
	}
	if got := spanAttributeValue(span.Attributes, "db.statement"); got == "" {
		t.Fatalf("db.statement should not be empty")
	}
	if span.Status.Code != otelcodes.Unset {
		t.Fatalf("db span status = %v, want unset", span.Status.Code)
	}
}

func TestObserverPluginRecordsDBSpanError(t *testing.T) {
	db := openObserverTestDB(t, fmt.Sprintf("orm-trace-error-%d", time.Now().UnixNano()))
	me := gztracetest.NewInMemoryExporter(t)

	ctx, parent := otel.Tracer("orm-test").Start(context.Background(), "parent")
	err := db.WithContext(ctx).Table("missing_table").Where("id = ?", 1).Take(&observerTestModel{}).Error
	if err == nil {
		t.Fatalf("expected missing table error")
	}
	parent.End()

	span := findSpanStubByName(t, me.GetSpans(), "db.query missing_table")
	if got := spanAttributeValue(span.Attributes, "db.table"); got != "missing_table" {
		t.Fatalf("db.table = %q, want missing_table", got)
	}
	if span.Status.Code != otelcodes.Error {
		t.Fatalf("db span status = %v, want error", span.Status.Code)
	}
	if span.Status.Description == "" {
		t.Fatalf("db span error description should not be empty")
	}
	if len(span.Events) == 0 {
		t.Fatalf("db span should record an error event")
	}
}

func metricCounterValue(t *testing.T, familyName string, labels map[string]string) float64 {
	t.Helper()

	families, err := metricGather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}

		for _, metric := range family.GetMetric() {
			if metricLabelsMatch(metric, labels) && metric.GetCounter() != nil {
				return metric.GetCounter().GetValue()
			}
		}
	}

	return 0
}

func metricHistogramCount(t *testing.T, familyName string, labels map[string]string) uint64 {
	t.Helper()

	families, err := metricGather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}

	for _, family := range families {
		if family.GetName() != familyName {
			continue
		}

		for _, metric := range family.GetMetric() {
			if metricLabelsMatch(metric, labels) && metric.GetHistogram() != nil {
				return metric.GetHistogram().GetSampleCount()
			}
		}
	}

	return 0
}

func metricGather() ([]*dto.MetricFamily, error) {
	return prometheus.DefaultGatherer.Gather()
}

func metricLabelsMatch(metric *dto.Metric, labels map[string]string) bool {
	if metric == nil {
		return false
	}

	for key, want := range labels {
		found := false
		for _, label := range metric.GetLabel() {
			if label.GetName() == key && label.GetValue() == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func openObserverTestDB(t *testing.T, service string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", service)), &gorm.Config{
		Logger: &observerLogger{level: logger.Silent},
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.Use(NewObserverPlugin(service, time.Hour)); err != nil {
		t.Fatalf("use observer plugin: %v", err)
	}
	if err := db.AutoMigrate(&observerTestModel{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

	return db
}

func findSpanStubByName(t *testing.T, spans oteltracetest.SpanStubs, name string) oteltracetest.SpanStub {
	t.Helper()

	for _, span := range spans {
		if span.Name == name {
			return span
		}
	}

	t.Fatalf("span %q not found in %+v", name, spans)
	return oteltracetest.SpanStub{}
}

func spanAttributeValue(attrs []attribute.KeyValue, key string) string {
	for _, attr := range attrs {
		if string(attr.Key) == key {
			return attr.Value.AsString()
		}
	}

	return ""
}
