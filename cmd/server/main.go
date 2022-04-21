package main

import (
	"path/filepath"

	"context"
	"github.com/gin-contrib/multitemplate"
	"go.uber.org/zap"
	"html/template"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var minioClient *minio.Client
var mediaBucket string
var thumbnailBucket string
var resourcesDir string
var useSSL bool

var initialPass string
var initialUser string

var DB *gorm.DB

var jwtKey []byte

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

var log *zap.SugaredLogger

func main() {

	logger, _ := zap.NewDevelopment()
	defer logger.Sync() // flushes buffer, if any
	log = logger.Sugar()

	var err error

	// S3 Connection parameters
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	mediaBucket = os.Getenv("S3_BUCKET_MEDIA")
	thumbnailBucket = os.Getenv("S3_BUCKET_THUMBNAILS")

	initialUser = os.Getenv("INITIAL_USER")
	initialPass = os.Getenv("INITIAL_PASS")

	useSSL, err = strconv.ParseBool(os.Getenv("S3_SSL"))
	if err != nil {
		log.Fatal("S3_SSL not set")
	}

	// JWT key
	if len(os.Getenv("JWT_KEY")) == 0 {
		log.Fatal("No JWT key set")
	}
	jwtKey = []byte(os.Getenv("JWT_KEY"))

	resourcesDir = os.Getenv("RESOURCES_DIR")
	if len(resourcesDir) == 0 {
		resourcesDir = "."
	}

	var db *gorm.DB

	// Setup database
	// TODO use an actual file for persistance
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
	initialPassHash, err := hashAndSalt(initialPass)
	if err != nil {
		log.Fatal(err)
	}

	_, _ = insertUser(initialUser, initialPassHash, true, 30)

	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Setup router
	r := gin.Default()

	// Load templates with custom renderer
	r.HTMLRender = loadTemplates(path.Join(resourcesDir, "templates"))
	// r.Delims("{[{", "}]}")

	// Set up routes

	// Routes accessible to anyone
	r.POST("/login", login)

	r.GET("/login", func(c *gin.Context) {
		c.HTML(http.StatusOK, "login.html", nil)
	})

	r.Static("/static", path.Join(resourcesDir, "static"))

	// Routes accessible to logged in users
	r.Use(verifyToken)
	r.GET("/", indexHandler)
	r.GET("/albums/:album", albumHandler)
	r.GET("/albums/:album/:image", imageHandler)

	// Routes accessible to admins only
	r.Use(verifyAdmin)
	r.GET("/me", getUserInfo) // TODO remove after testing
	r.GET("/users", getUsers)
	r.POST("/users", createUser)
	r.GET("/users/:user/delete", deleteUser)

	log.Info("starting gin")
	if err := r.Run("localhost:7788"); err != nil {
		log.Fatal(err)
	}
}

type templateData struct {
	Context *gin.Context
	Data    interface{}
}

func listObjectsByPrefix(prefix string) []string {
	log.Info("listing:", prefix)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// List objects
	objectCh := minioClient.ListObjects(ctx, mediaBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	ret := []string{}
	for object := range objectCh {
		if object.Err != nil {
			log.Error(object.Err)
			return ret
		}
		ret = append(ret, strings.TrimPrefix(strings.TrimSuffix(object.Key, "/"), prefix))
	}
	// log.Info(ret)
	return ret
}

func indexHandler(c *gin.Context) {

	td := templateData{

		Context: c,
		Data: struct {
			Title  string
			Albums []string
		}{
			Title:  "Albums",
			Albums: listObjectsByPrefix(c.GetString("username") + "/"),
		},
	}

	c.HTML(http.StatusOK, "index.html", td)
}
