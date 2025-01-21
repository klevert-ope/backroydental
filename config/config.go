package config

// AppConfig holds the application configuration
type AppConfig struct {
	DBURL        string
	RedisAddress string
	BearerToken  string
}

// GetBearerToken returns the BearerToken from the config
func (c *AppConfig) GetBearerToken() string {
	return c.BearerToken
}
