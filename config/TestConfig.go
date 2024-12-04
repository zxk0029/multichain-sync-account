package config

func DbConfigTest() *DBConfig {
	return &DBConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Name:     "multichain",
		User:     "postgres",
		Password: "123456",
	}
}
