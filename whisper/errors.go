package whisper

import (
	"errors"

	"github.com/jackc/pgx/v5"
)

// Whisper ORM hata tipleri.
// Kullanıcılar errors.Is(err, whisper.ErrNotFound) şeklinde kontrol edebilir.
var (
	// ErrNotFound sorgu sonucu boş döndüğünde kullanılır.
	ErrNotFound = errors.New("whisper: record not found")

	// ErrNoRows pgx.ErrNoRows'un Whisper sarmalayıcısıdır.
	ErrNoRows = pgx.ErrNoRows
)

// WrapScanError, pgx scan hatalarını anlamlı Whisper hatalarına çevirir.
// Eğer hata pgx.ErrNoRows ise ErrNotFound döner.
// Diğer hatalarda orijinal hatayı olduğu gibi döner.
func WrapScanError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
