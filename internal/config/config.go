package config

import (
	configwbf "github.com/wb-go/wbf/config"
)

type Config struct {
	Server  ServerConfig
	Kafka   KafkaConfig
	Postgre PostgreConfig
	Minio   MinioConfig
}

type MinioConfig struct {
	Endpoint   string
	User       string
	Password   string
	Sslmode    bool
	BucketName string
}

type ServerConfig struct {
	Port string
}

type KafkaConfig struct {
	Brokers []string
	Topic   string
	GroupID string
}

type PostgreConfig struct {
	User     string
	Password string
	Host     string
	DBName   string
}

func NewConfig(file string) (*Config, error) {
	c := configwbf.New()
	err := c.Load(file, "", "")
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: c.GetString("PORT"),
		},
		Postgre: PostgreConfig{
			Host:     c.GetString("POSTGRES_HOST"),
			DBName:   c.GetString("POSTGRES_DB"),
			User:     c.GetString("POSTGRES_USER"),
			Password: c.GetString("POSTGRES_PASSWORD"),
		},
		Kafka: KafkaConfig{
			Brokers: c.GetStringSlice("KAFKA_BROKERS"),
			Topic:   c.GetString("KAFKA_TOPIC"),
			GroupID: c.GetString("KAFKA_GROUP"),
		},
		Minio: MinioConfig{
			User:       c.GetString("MINIO_ROOT_USER"),
			Password:   c.GetString("MINIO_ROOT_PASSWORD"),
			Sslmode:    c.GetBool("MINIO_USE_SSL"),
			BucketName: c.GetString("MINIO_BUCKET_NAME"),
			Endpoint:   c.GetString("MINIO_ENDPOINT"),
		},
	}, nil
}
