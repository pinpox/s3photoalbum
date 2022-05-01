package s3photoalbum

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	S3Endpoint        string `split_words:"true" required:"true"`
	S3AccessKey       string `split_words:"true" required:"true"`
	S3SecretKey       string `split_words:"true" required:"true"`
	S3MediaBucket     string `split_words:"true" required:"true"`
	S3ThumbnailBucket string `split_words:"true" required:"true"`
	S3UseSsl          bool   `split_words:"true" default:"true"`
	ResourcesDir      string `split_words:"true" default:"."`
	ModeDevelop       bool   `split_words:"true" default:"false"`
	JwtKey            string `split_words:"true" required:"true"`
	InitialUser       string `split_words:"true" default:"admin"`
	InitialPass       string `split_words:"true" default:"admin"`
	Host              string `split_words:"true" defaulut:"localhost"`
	ListenAddress     string `split_words:"true" default:"127.0.0.1"`
	ListenPort        string `split_words:"true" default:"7788"`
}

func LoadConfig(path string) (config Config) {

	err := envconfig.Process("s3g", &config)
	if err != nil {
		panic(err.Error())
	}
	// fmt.Printf("%#v\n", config)
	return
}
