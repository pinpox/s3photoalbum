package s3photoalbum

import (
	"github.com/kelseyhightower/envconfig"
)

type CommonConfig struct {
	S3Endpoint        string `split_words:"true" required:"true"`
	S3AccessKey       string `split_words:"true" required:"true"`
	S3SecretKey       string `split_words:"true" required:"true"`
	S3MediaBucket     string `split_words:"true" required:"true"`
	S3ThumbnailBucket string `split_words:"true" required:"true"`
	S3UseSsl          bool   `split_words:"true" default:"true"`
	ModeDevelop       bool   `split_words:"true" default:"false"`
}

type ServerConfig struct {
	CommonConfig

	ResourcesDir  string `split_words:"true" default:"."`
	JwtKey        string `split_words:"true" required:"true"`
	InitialUser   string `split_words:"true" default:"admin"`
	InitialPass   string `split_words:"true" default:"admin"`
	Host          string `split_words:"true" default:"localhost"`
	ListenAddress string `split_words:"true" default:"127.0.0.1"`
	ListenPort    string `split_words:"true" default:"7788"`
}

type ThumbnailerConfig struct {
	CommonConfig
	ThumbnailSize         string `split_words:"true" default:"300"`
	FfmpegThumbnailerPath string `split_words:"true" required:"true"`
	ExifToolPath          string `split_words:"true" required:"true"`
}

func LoadServerConfig() (config ServerConfig) {
	err := envconfig.Process("s3g", &config)
	if err != nil {
		panic(err.Error())
	}
	return
}

func LoadThumbnailerConfig() (config ThumbnailerConfig) {
	err := envconfig.Process("s3g", &config)
	if err != nil {
		panic(err.Error())
	}
	return
}
