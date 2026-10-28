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
		g.GenerateModel("ran_feed_im_conversation"),
		g.GenerateModel("ran_feed_im_conversation_member"),
		g.GenerateModel("ran_feed_im_message"),
		g.GenerateModel("ran_feed_im_group"),
	)

	g.Execute()
}
