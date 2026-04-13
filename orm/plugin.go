package orm

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	gztrace "github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/gorm"
)

type observerPlugin struct {
	service       string
	slowThreshold time.Duration
}

func NewObserverPlugin(service string, slowThreshold time.Duration) gorm.Plugin {
	return &observerPlugin{
		service:       normalizeService(service),
		slowThreshold: slowThreshold,
	}
}

func (p *observerPlugin) Name() string {
	return "zfeed-observer-plugin"
}

func (p *observerPlugin) Initialize(db *gorm.DB) error {
	if err := registerCreateCallbacks(db, p.observe); err != nil {
		return err
	}
	if err := registerQueryCallbacks(db, p.observe); err != nil {
		return err
	}
	if err := registerUpdateCallbacks(db, p.observe); err != nil {
		return err
	}
	if err := registerDeleteCallbacks(db, p.observe); err != nil {
		return err
	}
	if err := registerRowCallbacks(db, p.observe); err != nil {
		return err
	}
	return registerRawCallbacks(db, p.observe)
}

func (p *observerPlugin) observe(db *gorm.DB, operation string) {
	start, ok := db.InstanceGet(observerStartKey(operation))
	if !ok {
		return
	}

	startTime, ok := start.(time.Time)
	if !ok {
		return
	}

	elapsed := time.Since(startTime)
	table := extractTable(db)
	result := classifyResult(db.Error)
	service := normalizeService(p.service)
	statement := compactSQL(db.Statement.SQL.String())

	metricStatementDuration.Observe(elapsed.Milliseconds(), service, operation, table)
	metricStatementTotal.Inc(service, operation, table, result)
	if elapsed >= p.slowThreshold {
		metricStatementSlowTotal.Inc(service, operation, table)
	}
	finishTraceSpan(db, operation, table, result, statement)

	writeStatementLog(statementLog{
		ctx:           extractContext(db),
		service:       service,
		operation:     operation,
		table:         table,
		result:        result,
		statement:     statement,
		rows:          db.RowsAffected,
		elapsed:       elapsed,
		err:           db.Error,
		slowThreshold: p.slowThreshold,
	})
}

func registerCreateCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Create()
	operation := "create"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation
	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}
	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func registerQueryCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Query()
	operation := "query"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation
	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}
	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func registerUpdateCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Update()
	operation := "update"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation
	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}
	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func registerDeleteCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Delete()
	operation := "delete"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation
	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}
	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func registerRowCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Row()
	operation := "row"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation
	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}
	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func registerRawCallbacks(db *gorm.DB, observer func(*gorm.DB, string)) error {
	processor := db.Callback().Raw()
	operation := "raw"
	beforeName := "zfeed:observer:before:" + operation
	afterName := "zfeed:observer:after:" + operation

	if err := processor.Before("*").Register(beforeName, func(db *gorm.DB) {
		beginObserve(db, operation)
	}); err != nil {
		return err
	}

	return processor.After("*").Register(afterName, func(db *gorm.DB) {
		observer(db, operation)
	})
}

func observerStartKey(operation string) string {
	return "zfeed:observer:start:" + operation
}

func observerSpanKey(operation string) string {
	return "zfeed:observer:span:" + operation
}

func beginObserve(db *gorm.DB, operation string) {
	db.InstanceSet(observerStartKey(operation), time.Now())

	ctx := extractContext(db)
	tracer := gztrace.TracerFromContext(ctx)
	_, span := tracer.Start(ctx, buildDBSpanName(operation, extractTable(db)),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient))
	db.InstanceSet(observerSpanKey(operation), span)
}

func finishTraceSpan(db *gorm.DB, operation, table, result, statement string) {
	value, ok := db.InstanceGet(observerSpanKey(operation))
	if !ok {
		return
	}

	span, ok := value.(oteltrace.Span)
	if !ok || span == nil {
		return
	}
	defer span.End()

	span.SetName(buildDBSpanName(operation, table))
	span.SetAttributes(
		attribute.String("db.system", "mysql"),
		attribute.String("db.operation", operation),
		attribute.String("db.table", table),
		attribute.String("zfeed.db.result", result),
		attribute.Int64("db.rows_affected", db.RowsAffected),
	)
	if statement != "" {
		span.SetAttributes(attribute.String("db.statement", statement))
	}

	if db.Error == nil || errors.Is(db.Error, gorm.ErrRecordNotFound) {
		return
	}

	span.RecordError(db.Error)
	span.SetStatus(otelcodes.Error, db.Error.Error())
}

func buildDBSpanName(operation, table string) string {
	if table == "" || table == "unknown" {
		return "db." + operation
	}
	return "db." + operation + " " + table
}

func classifyResult(err error) string {
	switch {
	case err == nil:
		return "ok"
	case errors.Is(err, gorm.ErrRecordNotFound):
		return "not_found"
	default:
		return "error"
	}
}

func normalizeService(service string) string {
	service = strings.TrimSpace(service)
	if service == "" {
		return "unknown"
	}
	return service
}

func extractContext(db *gorm.DB) context.Context {
	if db == nil || db.Statement == nil || db.Statement.Context == nil {
		return context.Background()
	}
	return db.Statement.Context
}

func extractTable(db *gorm.DB) string {
	if db != nil && db.Statement != nil {
		if table := strings.TrimSpace(db.Statement.Table); table != "" {
			return table
		}
		if schema := db.Statement.Schema; schema != nil && strings.TrimSpace(schema.Table) != "" {
			return schema.Table
		}
		if raw := compactSQL(db.Statement.SQL.String()); raw != "" {
			if table := tableFromSQL(raw); table != "" {
				return table
			}
		}
	}
	return "unknown"
}

var tablePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bfrom\s+[` + "`" + `"]?([a-zA-Z0-9_\.]+)[` + "`" + `"]?`),
	regexp.MustCompile(`(?i)\binto\s+[` + "`" + `"]?([a-zA-Z0-9_\.]+)[` + "`" + `"]?`),
	regexp.MustCompile(`(?i)\bupdate\s+[` + "`" + `"]?([a-zA-Z0-9_\.]+)[` + "`" + `"]?`),
	regexp.MustCompile(`(?i)\bdelete\s+from\s+[` + "`" + `"]?([a-zA-Z0-9_\.]+)[` + "`" + `"]?`),
}

func tableFromSQL(stmt string) string {
	for _, pattern := range tablePatterns {
		matches := pattern.FindStringSubmatch(stmt)
		if len(matches) == 2 {
			return strings.Trim(matches[1], "`\" ")
		}
	}
	return ""
}

func compactSQL(stmt string) string {
	stmt = strings.TrimSpace(stmt)
	if stmt == "" {
		return ""
	}
	stmt = strings.Join(strings.Fields(stmt), " ")
	if len(stmt) > 512 {
		return stmt[:512] + "...(truncated)"
	}
	return stmt
}

type statementLog struct {
	ctx           context.Context
	service       string
	operation     string
	table         string
	result        string
	statement     string
	rows          int64
	elapsed       time.Duration
	err           error
	slowThreshold time.Duration
}

func writeStatementLog(entry statementLog) {
	logger := logx.WithContext(entry.ctx).WithDuration(entry.elapsed)
	fields := []logx.LogField{
		logx.Field("layer", "db"),
		logx.Field("service", entry.service),
		logx.Field("operation", entry.operation),
		logx.Field("table", entry.table),
		logx.Field("result", entry.result),
		logx.Field("rows", entry.rows),
		logx.Field("statement", entry.statement),
	}

	switch {
	case entry.err != nil && !errors.Is(entry.err, gorm.ErrRecordNotFound):
		fields = append(fields, logx.Field("error", entry.err.Error()))
		logger.Errorw("db.statement", fields...)
	case entry.elapsed >= entry.slowThreshold:
		logger.Sloww("db.statement", fields...)
	default:
		logger.Infow("db.statement", fields...)
	}
}
