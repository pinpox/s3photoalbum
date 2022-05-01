package main

import (
	"path/filepath"
	"s3photoalbum/internal"

	"github.com/gin-contrib/multitemplate"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"path"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	minioClient *minio.Client
	config      s3photoalbum.ServerConfig
	DB          *gorm.DB
	log         *zap.SugaredLogger
)

func loadTemplates(templatesDir string) multitemplate.Renderer {
	r := multitemplate.NewRenderer()

	funcmap := template.FuncMap{
		"incolumn":    func(colNum, index int) bool { return index%4 == colNum },
		"isLoggedIn":  func(c *gin.Context) bool { return c.GetString("username") != "" },
		"getUsername": func(c *gin.Context) string { return c.GetString("username") },
		"isAdmin":     func(c *gin.Context) bool { return c.GetBool("isadmin") },
	}

	// Read all partials, they will be appended to all templates
	partials, err := filepath.Glob(path.Join(templatesDir, "partials", "*.html"))
	if err != nil {
		log.Fatal(err.Error())
	}

	// Read all templates
	templates, err := filepath.Glob(path.Join(templatesDir, "/*.html"))
	if err != nil {
		log.Fatal(err.Error())
	}

	// Add templates, naming them by their basename
	for _, template := range templates {
		templList := append([]string{}, template)
		templList = append(templList, partials...)
		r.AddFromFilesFuncs(filepath.Base(template), funcmap, templList...)
	}

	return r
}

func main() {

	var err error

	config = s3photoalbum.LoadServerConfig()
	log = s3photoalbum.NewLogger(config.ModeDevelop)

	var db *gorm.DB

	// Setup database
	// db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	db, err = gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
	if err != nil {
		log.Fatal(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal(err)
	}
	DB = db

	// TODO improve intial user creation, check for existing
	initialPassHash, err := hashAndSalt(config.InitialPass)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = insertUser(config.InitialUser, initialPassHash, true)

	// Initialize minio client object.
	minioClient, err = minio.New(config.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.S3AccessKey, config.S3SecretKey, ""),
		Secure: config.S3UseSsl,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Setup router
	r := gin.Default()

	// Load templates with custom renderer
	r.HTMLRender = loadTemplates(path.Join(config.ResourcesDir, "templates"))

	// Set up routes

	// Routes accessible to anyone
	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", gin.H{
			"context": c,
			"title":   "Login",
		})
	})
	r.POST("/login", login)
	r.Static("/static", path.Join(config.ResourcesDir, "static"))

	// Routes accessible to logged in users
	r.Use(verifyToken)
	r.GET("/", indexHandler)
	r.GET("/albums/:album", albumHandler)
	r.GET("/albums/:album/:image", imageHandler)
	r.GET("/thumbnails/:album/:image", thumbnailHandler)

	// Routes accessible to admins only
	r.Use(verifyAdmin)
	r.GET("/me", getUserInfo) // TODO remove after testing
	r.GET("/users", getUsers)
	r.POST("/users", createUser)
	r.GET("/users/:user/delete", deleteUser)

	log.Info("starting gin")
	if err := r.Run(config.ListenAddress + ":" + config.ListenPort); err != nil {
		log.Fatal(err)
	}
}

func indexHandler(c *gin.Context) {

	albums, err := getAlbumsByUsername(c.GetString("username"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":   "Albums",
		"albums":  albums,
		"context": c,
	})
}
