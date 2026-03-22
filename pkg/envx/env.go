package envx

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/logx"
)

const defaultEnvFile = ".env"

// Load tries to load .env for local dev. It will not fail if the file is missing.
// If ENV_FILE is set, it will try that file and log errors but continue.
func Load() {
	envFile := os.Getenv("ENV_FILE")
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			logx.Errorf("加载环境变量文件 %s 失败: %v", envFile, err)
		}
		return
	}

	if err := godotenv.Load(defaultEnvFile); err != nil {
		logx.Errorf("加载环境变量文件 %s 失败: %v", defaultEnvFile, err)
	}
}

// MustLoad loads environment variables and exits on failure.
func MustLoad() {
	envFile := os.Getenv("ENV_FILE")
	if envFile != "" {
		if err := godotenv.Load(envFile); err != nil {
			logx.Errorf("加载环境变量文件 %s 失败: %v", envFile, err)
			os.Exit(1)
		}
		return
	}

	if err := godotenv.Load(defaultEnvFile); err != nil {
		logx.Errorf("加载环境变量文件 %s 失败: %v", defaultEnvFile, err)
		os.Exit(1)
	}
}

func GetenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func MySQLDSNFromEnv() string {
	host := GetenvDefault("MYSQL_HOST", "127.0.0.1")
	port := GetenvDefault("MYSQL_PORT", "3306")
	user := GetenvDefault("MYSQL_APP_USER", "root")
	pass := GetenvDefault("MYSQL_APP_PASSWORD", "root")
	dbName := GetenvDefault("MYSQL_DATABASE", "ran-feed")
	loc := GetenvDefault("MYSQL_LOC", "Asia%2FShanghai")

	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=true&loc=%s",
		user,
		pass,
		host,
		port,
		dbName,
		loc,
	)
}
