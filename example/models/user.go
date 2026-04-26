package models

import "time"

// User veritabanındaki kullanıcı tablosunu temsil eder
type User struct {
	ID        int       `db:"user_id,primary_key"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	Password  string    `db:"password_hash"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`

	// İlişkisel alan (Veritabanında kolon değil, Whisper ORM için bir ilişki)
	Products []Product `db:"-" whisper:"has_many:Product,fk:user_id"`
}

// Product bir ürün temsilidir
type Product struct {
	ID     int     `db:"id,primary_key"`
	UserID int     `db:"user_id"` // User tablosuna foreign key
	Name   string  `db:"name"`
	Price  float64 `db:"price"`
}
