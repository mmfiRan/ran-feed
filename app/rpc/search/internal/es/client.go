package es

import (
	"github.com/elastic/go-elasticsearch/v8"
)

// MustNewClient 创建 ES 客户端 失败直接 panic 由 supervisor 拉起
func MustNewClient(addresses []string, username, password string) *elasticsearch.Client {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: addresses,
		Username:  username,
		Password:  password,
	})
	if err != nil {
		panic(err)
	}
	return client
}
