package config

import (
	"github.com/caarlos0/env/v9"
	"github.com/joho/godotenv"
	"log"
	"os"
)

type Config struct {
	Mode        string `env:"MODE" envDefault:"dev"`
	ServiceName string `env:"SERVICE_NAME" envDefault:"sso"`
	Server      ServerConfig
	Auth        authConfig
	Email       smtpConfig
	DB          dbConfig
	Minio       s3Config
	Redis       redisConfig
	Jaeger      jaegerConfig
}

type ServerConfig struct {
	Scheme   string `env:"SERVER_SCHEME" envDefault:"http"`
	Domain   string `env:"SERVER_DOMAIN" envDefault:"localhost"`
	Port     int    `env:"SERVER_HTTP_PORT,required"`
	GRPCPort int    `env:"SERVER_GRPC_PORT" envDefault:"50050"`
	PromPort int    `env:"SERVER_PROM_PORT" envDefault:"8085"`
}

type authConfig struct {
	JWT struct {
		Secret string `env:"JWT_SECRET,required"`
		Issuer string `env:"JWT_ISSUER,required"`
	}
	Captcha struct {
		SiteKey string `env:"CAPTCHA_SITE_KEY"`
		Secret  string `env:"CAPTCHA_SECRET"`
	}
}

type smtpConfig struct {
	Server string `env:"EMAIL_SERVER" envDefault:"smtp.gmail.com"`
	Port   int    `env:"EMAIL_PORT" envDefault:"587"`
	User   string `env:"EMAIL_USER" envDefault:""`
	Pass   string `env:"EMAIL_PASS" envDefault:""`
	Admin  string `env:"EMAIL_ADMIN" envDefault:""`
}

type dbConfig struct {
	Host     string `env:"POSTGRES_HOST" envDefault:"localhost"`
	Port     int    `env:"POSTGRES_PORT" envDefault:"5432"`
	User     string `env:"POSTGRES_USER" envDefault:"app_owner"`
	Password string `env:"POSTGRES_PASSWORD" envDefault:"password"`
	Database string `env:"POSTGRES_DB" envDefault:"app_db"`
}

type s3Config struct {
	Addr      string `env:"MINIO_ADDR" envDefault:"localhost:9000"`
	AccessKey string `env:"MINIO_ACCESS_KEY" envDefault:""`
	SecretKey string `env:"MINIO_SECRET_KEY" envDefault:""`
	Bucket    string `env:"MINIO_BUCKET" envDefault:"app"`
	UseSSL    bool   `env:"MINIO_SSL" envDefault:"false"`
}

type redisConfig struct {
	Addr string `env:"REDIS_ADDR" envDefault:"localhost:6379"`
	Pass string `env:"REDIS_PASS" envDefault:""`
}

type jaegerConfig struct {
	Sampler struct {
		Type  string  `env:"JAEGER_SAMPLER_TYPE" envDefault:"const"`
		Param float64 `env:"JAEGER_SAMPLER_PARAM" envDefault:"1"`
	} `yaml:"sampler"`
	Reporter struct {
		LogSpans           bool   `env:"JAEGER_REPORTER_LOGSPANS" envDefault:"true"`
		LocalAgentHostPort string `env:"JAEGER_REPORTER_LOCALAGENT" envDefault:"localhost:6831"`
		CollectorEndpoint  string `env:"JAEGER_REPORTER_COLLECTOR" envDefault:"http://localhost:14268/api/traces"`
	} `yaml:"reporter"`
}

func MustLoad(path string) Config {
	if err := godotenv.Load(path); err != nil {
		if !os.IsNotExist(err) {
			panic("failed to load .env file: " + err.Error())
		}
		log.Println("No .env file found, using system environment variables")
	} else {
		log.Println("Loaded environment variables from: " + path)
	}

	conf := Config{}
	if err := env.Parse(&conf); err != nil {
		panic("failed to parse environment variables: " + err.Error())
	}
	log.Println("Load configuration from environment")

	return conf
}
