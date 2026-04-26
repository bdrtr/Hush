package whisper

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Queryable defines the common interface satisfied by both *pgxpool.Pool and pgx.Tx.
// This allows all ORM methods to accept either a connection pool or a transaction
// without any code duplication and with near-zero interface dispatch overhead.
type Queryable interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

// bufferPool sıfır tahsisat (zero-allocation) için string birleştirme havuzu sağlar.
var bufferPool = sync.Pool{
	New: func() interface{} {
		var b strings.Builder
		b.Grow(256)
		return &b
	},
}

// DB, pgxpool.Pool'un Whisper ORM sarmalayıcısıdır (wrapper).
type DB struct {
	Pool *pgxpool.Pool
}

// NewDB standart bir pgxpool bağlantısını alıp Whisper ORM'e dönüştürür.
func NewDB(pool *pgxpool.Pool) *DB {
	return &DB{Pool: pool}
}

// QueryBuilder zincirleme (chained) API sağlayan yapıdır.
type QueryBuilder struct {
	db        *DB
	selects   []string
	table     string
	wheres    []string
	args      []interface{}
	orderBy   string
	limitVal  int
	offsetVal int
}

// Select zinciri başlatır. (Örn: db.Select("id", "name"))
func (db *DB) Select(columns ...string) *QueryBuilder {
	if len(columns) == 0 {
		columns = []string{"*"}
	}
	return &QueryBuilder{
		db:       db,
		selects:  columns,
		limitVal: -1,
		offsetVal: -1,
	}
}

// From tabloyu belirler.
func (qb *QueryBuilder) From(table string) *QueryBuilder {
	qb.table = table
	return qb
}

// Where şartları ekler. AND ile birbirine bağlanır.
func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	qb.wheres = append(qb.wheres, condition)
	qb.args = append(qb.args, args...)
	return qb
}

// OrderBy sıralama ekler. (Örn: .OrderBy("created_at DESC"))
func (qb *QueryBuilder) OrderBy(clause string) *QueryBuilder {
	qb.orderBy = clause
	return qb
}

// Limit sorgu sonuç sayısını sınırlar.
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limitVal = n
	return qb
}

// Offset sorgu sonuçlarını atlar (Sayfalama için).
func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offsetVal = n
	return qb
}

// Build arka planda havuz kullanarak sorguyu (query) ve argümanları oluşturur.
func (qb *QueryBuilder) Build() (string, []interface{}) {
	b := bufferPool.Get().(*strings.Builder)
	b.Reset()
	
	// SELECT
	b.WriteString("SELECT ")
	b.WriteString(strings.Join(qb.selects, ", "))
	
	// FROM
	b.WriteString(" FROM ")
	b.WriteString(qb.table)
	
	// WHERE
	if len(qb.wheres) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(qb.wheres, " AND "))
	}

	// ORDER BY
	if qb.orderBy != "" {
		b.WriteString(" ORDER BY ")
		b.WriteString(qb.orderBy)
	}

	// LIMIT
	if qb.limitVal >= 0 {
		b.WriteString(" LIMIT ")
		b.WriteString(strconv.Itoa(qb.limitVal))
	}

	// OFFSET
	if qb.offsetVal >= 0 {
		b.WriteString(" OFFSET ")
		b.WriteString(strconv.Itoa(qb.offsetVal))
	}

	query := b.String()
	bufferPool.Put(b)
	
	return query, qb.args
}

// QueryRow tek bir satır döner. Opsiyonel olarak farklı bir Queryable geçilebilir.
func (qb *QueryBuilder) QueryRow(q ...Queryable) pgx.Row {
	var conn Queryable = qb.db.Pool
	if len(q) > 0 && q[0] != nil {
		conn = q[0]
	}
	query, args := qb.Build()
	return conn.QueryRow(context.Background(), query, args...)
}

// Query birden çok satır döner. Opsiyonel olarak farklı bir Queryable geçilebilir.
func (qb *QueryBuilder) Query(q ...Queryable) (pgx.Rows, error) {
	var conn Queryable = qb.db.Pool
	if len(q) > 0 && q[0] != nil {
		conn = q[0]
	}
	query, args := qb.Build()
	return conn.Query(context.Background(), query, args...)
}

// Count sorgu sonucunun toplam sayısını döner (Sayfalama için).
func (qb *QueryBuilder) Count(q ...Queryable) (int64, error) {
	var conn Queryable = qb.db.Pool
	if len(q) > 0 && q[0] != nil {
		conn = q[0]
	}

	b := bufferPool.Get().(*strings.Builder)
	b.Reset()
	b.WriteString("SELECT COUNT(*) FROM ")
	b.WriteString(qb.table)
	if len(qb.wheres) > 0 {
		b.WriteString(" WHERE ")
		b.WriteString(strings.Join(qb.wheres, " AND "))
	}
	query := b.String()
	bufferPool.Put(b)

	var count int64
	err := conn.QueryRow(context.Background(), query, qb.args...).Scan(&count)
	return count, err
}
