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
		g.GenerateModel("ran_feed_admin_user"),
		g.GenerateModel("ran_feed_admin_role"),
		g.GenerateModel("ran_feed_admin_permission"),
		g.GenerateModel("ran_feed_admin_user_role"),
		g.GenerateModel("ran_feed_admin_role_permission"),
		g.GenerateModel("ran_feed_operation_log"),
	)

	g.Execute()
}
