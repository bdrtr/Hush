package whisper

import (
	"context"
	"reflect"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

// columnInfo tablonun kolon bilgilerini saklar
type columnInfo struct {
	Name       string
	FieldIndex int
}

// tableInfo startup'ta bir kez analiz edilip önbelleğe alınan (cache) tablo şemasıdır
type tableInfo struct {
	Name    string
	Columns []columnInfo
	PKName  string
}

// tableCache reflection maliyetini startup'a (ilk çağrıya) indirgeyen sync.Map
var tableCache sync.Map // map[reflect.Type]*tableInfo

// getTableInfo gönderilen tipin (Type) şemasını okur veya Cache'ten getirir
func getTableInfo(t reflect.Type) *tableInfo {
	if val, ok := tableCache.Load(t); ok {
		return val.(*tableInfo)
	}

	info := &tableInfo{
		Name: strings.ToLower(t.Name()) + "s",
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		dbTag := field.Tag.Get("db")
		if dbTag == "" || dbTag == "-" {
			continue
		}
		
		parts := strings.Split(dbTag, ",")
		colName := parts[0]
		if colName == "" {
			colName = strings.ToLower(field.Name)
		}
		
		info.Columns = append(info.Columns, columnInfo{
			Name:       colName,
			FieldIndex: i,
		})

		for _, p := range parts {
			if p == "primary_key" {
				info.PKName = colName
			}
		}
	}
	
	if info.PKName == "" && len(info.Columns) > 0 {
		info.PKName = info.Columns[0].Name
	}

	tableCache.Store(t, info)
	return info
}

// Scanner, Code Generator tarafından üretilen modellerin Interface'idir
type Scanner interface {
	ScanRow(row pgx.Row) error
}

// buildSelectQuery, bir T tipi için SELECT sorgusunu buffer pool kullanarak oluşturur.
func buildSelectQuery(info *tableInfo) string {
	b := bufferPool.Get().(*strings.Builder)
	b.Reset()

	b.WriteString("SELECT ")
	for i, c := range info.Columns {
		b.WriteString(c.Name)
		if i < len(info.Columns)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(" FROM ")
	b.WriteString(info.Name)

	query := b.String()
	bufferPool.Put(b)
	return query
}

// Find, Go Generics kullanarak ID ile tek kayıt çeker. (Type-Safe)
// Kayıt bulunamazsa whisper.ErrNotFound döner.
// Örnek: user, err := whisper.Find[User](ctx, pool, 1)
func Find[T any](ctx context.Context, q Queryable, id interface{}) (*T, error) {
	var model T
	t := reflect.TypeOf(model)
	info := getTableInfo(t)

	b := bufferPool.Get().(*strings.Builder)
	b.Reset()
	b.WriteString(buildSelectQuery(info))
	b.WriteString(" WHERE ")
	b.WriteString(info.PKName)
	b.WriteString(" = $1")
	query := b.String()
	bufferPool.Put(b)
	
	row := q.QueryRow(ctx, query, id)

	// FAST PATH: Code-Gen ile üretilmişse reflection'ı atla
	if scanner, ok := any(&model).(Scanner); ok {
		err := WrapScanError(scanner.ScanRow(row))
		return &model, err
	}

	// SLOW PATH: Dinamik reflection
	v := reflect.ValueOf(&model).Elem()
	scanArgs := make([]interface{}, len(info.Columns))
	for i, col := range info.Columns {
		scanArgs[i] = v.Field(col.FieldIndex).Addr().Interface()
	}

	err := WrapScanError(row.Scan(scanArgs...))
	return &model, err
}

// FindAll, bir tablodaki tüm kayıtları çeker. (Type-Safe)
// Örnek: users, err := whisper.FindAll[User](ctx, pool)
func FindAll[T any](ctx context.Context, q Queryable) ([]T, error) {
	var model T
	t := reflect.TypeOf(model)
	info := getTableInfo(t)

	query := buildSelectQuery(info)

	rows, err := q.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsGeneric[T](rows, info)
}

// FindWhere, WHERE koşulu ile kayıtları çeker. (Type-Safe)
// Örnek: users, err := whisper.FindWhere[User](ctx, pool, "is_active = $1", true)
func FindWhere[T any](ctx context.Context, q Queryable, condition string, args ...interface{}) ([]T, error) {
	var model T
	t := reflect.TypeOf(model)
	info := getTableInfo(t)

	b := bufferPool.Get().(*strings.Builder)
	b.Reset()
	b.WriteString(buildSelectQuery(info))
	b.WriteString(" WHERE ")
	b.WriteString(condition)
	query := b.String()
	bufferPool.Put(b)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanRowsGeneric[T](rows, info)
}

// scanRowsGeneric, pgx.Rows'u generic T slice'a çevirir.
// FAST PATH: Eğer T, Scanner interface'ini sağlıyorsa Code-Gen'i kullanır.
// SLOW PATH: Reflection ile dinamik atama yapar.
func scanRowsGeneric[T any](rows pgx.Rows, info *tableInfo) ([]T, error) {
	var results []T

	for rows.Next() {
		var model T

		// FAST PATH
		if scanner, ok := any(&model).(Scanner); ok {
			if err := scanner.ScanRow(rows); err != nil {
				return nil, err
			}
			results = append(results, model)
			continue
		}

		// SLOW PATH
		v := reflect.ValueOf(&model).Elem()
		scanArgs := make([]interface{}, len(info.Columns))
		for i, col := range info.Columns {
			scanArgs[i] = v.Field(col.FieldIndex).Addr().Interface()
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		results = append(results, model)
	}

	return results, nil
}
