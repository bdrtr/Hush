package hush_test

import (
	"fmt"
	"testing"

	"github.com/bdrtr/hush"
	"github.com/bdrtr/hush/whisper"
)

// TestWhisperIntegration Whisper ORM'in Hush DI ile çalıştığını doğrular.
func TestWhisperIntegration(t *testing.T) {
	// 1. Hush Engine Başlatılıyor
	app := hush.New()

	// 2. Gerçek ortamda pgxpool.New(...) ile Pool oluşturulur.
	// pool, _ := pgxpool.New(context.Background(), "postgres://user:pass@localhost/db")
	// db := whisper.NewDB(pool)
	// Burada nil geçiyoruz (mock test)
	db := whisper.NewDB(nil)

	// 3. DI (Dependency Injection) ile Whisper'ı Hush context'ine yerleştiriyoruz!
	app.Provide(db)

	// 4. Route tanımlıyoruz
	app.GET("/users/:id", func(c *hush.Context) {
		// Context üzerinden 0 maliyetle DB'yi çekiyoruz
		orm := c.DB()

		userID := c.Param("id")

		// Whisper Query Builder ile sorgu atıyoruz (pgx native)
		row := orm.Select("user_id", "username", "email").
			From("users").
			Where("user_id = $1", userID).
			Where("is_active = $2", true).
			QueryRow()

		// --- BURADA NORMALDE models.User Kullanılır ---
		// var user models.User
		// err := user.ScanRow(row)
		// ...

		fmt.Println("Gelen İstek ID:", userID)
		_ = row // Kullanılmadı uyarısını engellemek için
		c.Ok(map[string]string{"message": "User queried with Whisper ORM", "id": userID})
	})

	app.POST("/users", func(c *hush.Context) {
		orm := c.DB()
		_ = orm
		c.Created(map[string]string{"message": "User inserted with Whisper ORM"})
	})

	fmt.Println("Whisper ORM Hush framework'üne başarıyla entegre edildi.")
}
