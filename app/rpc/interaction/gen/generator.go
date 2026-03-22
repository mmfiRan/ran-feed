package main

import (
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"

	"ran-feed/pkg/envx"
)

func main() {
	envx.Load()

	g := gen.NewGenerator(gen.Config{
		OutPath:       "./internal/entity/query",
		Mode:          gen.WithDefaultQuery | gen.WithQueryInterface,
		FieldNullable: true,
	})

	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = envx.MySQLDSNFromEnv()
	}

	db, err := gorm.Open(mysql.Open(dsn))
	if err != nil {
		panic(err)
	}

	g.UseDB(db)

	g.ApplyBasic(
		g.GenerateModel("ran_feed_comment"),
		g.GenerateModel("ran_feed_favorite"),
		g.GenerateModel("ran_feed_like"),
		g.GenerateModel("ran_feed_follow"),
		g.GenerateModel("ran_feed_mq_consume_dedup"),
	)

	g.Execute()
}
