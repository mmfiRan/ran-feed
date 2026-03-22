package xxljob

import "time"

type Config struct {
	AppName      string
	ExecutorName string
	Address      string
	IP           string
	Port         int
	AccessToken  string

	AdminAddresses   []string
	RegistryInterval time.Duration
	HTTPTimeout      time.Duration
}

func (c *Config) normalize() {
	if c.RegistryInterval <= 0 {
		c.RegistryInterval = 30 * time.Second
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 10 * time.Second
	}
}
