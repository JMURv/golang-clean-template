package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	ServiceName string        `yaml:"serviceName" env-required:"true"`
	Secret      string        `yaml:"secret" env-required:"true"`
	Server      *ServerConfig `yaml:"server"`
	DB          *DBConfig     `yaml:"db"`
	Redis       *RedisConfig  `yaml:"redis"`
	Jaeger      *JaegerConfig `yaml:"jaeger"`
}

type ServerConfig struct {
	Mode   string `yaml:"mode" env-default:"dev"`
	Port   int    `yaml:"port" env-required:"true"`
	Scheme string `yaml:"scheme" env-default:"http"`
	Domain string `yaml:"domain" env-default:"localhost"`
}

type DBConfig struct {
	Host     string `yaml:"host" env-default:"localhost"`
	Port     int    `yaml:"port" env-default:"5432"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
	Database string `yaml:"database" env-default:"db"`
}

type RedisConfig struct {
	Addr string `yaml:"addr" env-default:"localhost:6379"`
	Pass string `yaml:"pass" env-default:""`
}

type JaegerConfig struct {
	Sampler struct {
		Type  string `yaml:"type"`
		Param int    `yaml:"param"`
	} `yaml:"sampler"`
	Reporter struct {
		LogSpans           bool   `yaml:"LogSpans"`
		LocalAgentHostPort string `yaml:"LocalAgentHostPort"`
		CollectorEndpoint  string `yaml:"CollectorEndpoint"`
	} `yaml:"reporter"`
}

func MustLoad(configPath string) *Config {
	conf := &Config{}

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic("failed to read config: " + err.Error())
	}

	if err = yaml.Unmarshal(data, conf); err != nil {
		panic("failed to unmarshal config: " + err.Error())
	}

	return conf
}
