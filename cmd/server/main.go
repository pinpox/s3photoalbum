package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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

func main() {

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
	db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	// db, err = gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		panic(err)
	}
	DB = db

	// TODO improve intial user creation, check for existing
	initialPassHash, err := hashAndSalt(initialPass)
	if err != nil {
		log.Fatalln(err)
	}

	_, _ = insertUser(initialUser,initialPassHash, true, 30)

	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Setup router
	r := gin.Default()

	// Load templates
	// r.Delims("{[{", "}]}")
	r.SetFuncMap(template.FuncMap{
		"incolumn": func(colNum, index int) bool { return index%4 == colNum },
	})

	r.LoadHTMLGlob(path.Join(resourcesDir, "templates", "*.html"))

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

	fmt.Println("starting gin")
	if err := r.Run("localhost:7788"); err != nil {
		panic(err)
	}
}

func deleteUser(c *gin.Context) {
	formUser := c.Param("user")

	result := DB.Delete(&User{}, formUser)
	if result.Error != nil {
			fmt.Println(result.Error)
	}

	c.Redirect(http.StatusSeeOther, "/users")
}

func createUser(c *gin.Context) {

	formUser := c.PostForm("username")
	formPass := c.PostForm("password")
	formIsAdmin := c.PostForm("isadmin")
	formAge := c.PostForm("age")

	passwordHash, err := hashAndSalt(formPass)
	if err != nil {
		fmt.Println("failed to hash pass", err)
		getUsers(c)
	}

	userAge, err := strconv.ParseUint(formAge, 10, 64)
	if err != nil {
		fmt.Println("failed to convert age", err)
		getUsers(c)
	}

	_, err = insertUser(formUser, passwordHash, formIsAdmin == "on", uint(userAge))
	if err != nil {
		fmt.Println("failed to insert user", err)
		getUsers(c)
	}

	c.Redirect(http.StatusSeeOther, "/users")

}

func getUsers(c *gin.Context) {

	var users []User

	result := DB.Find(&users)
	if result.Error != nil {
		panic(result.Error)
	}

	c.HTML(http.StatusOK, "users.html", users)
}

func albumHandler(c *gin.Context) {

	ad := struct {
		Title  string
		Images []string
	}{
		Title:  c.Param("album"),
		Images: listObjectsByPrefix(c.Param("album") + "/"),
	}

	c.HTML(http.StatusOK, "album.html", ad)
}

func imageHandler(c *gin.Context) {

	res := c.DefaultQuery("thumbnail", "false")
	thumbnail, err := strconv.ParseBool(res)
	if err != nil {
		thumbnail = false
	}

	imgPath := c.Param("album") + "/" + c.Param("image")

	// Set request parameters for content-disposition.
	reqParams := make(url.Values)

	// TODO for download
	// reqParams.Set("response-content-disposition", "attachment; filename=\""+ps.ByName("image")+"\"")

	var presignedURL *url.URL

	if thumbnail {

		thumbPath := imgPath + ".jpg"

		objInfo, err := minioClient.StatObject(context.Background(), thumbnailBucket, thumbPath, minio.StatObjectOptions{})
		if err != nil {

			errResponse := minio.ToErrorResponse(err)
			if errResponse.Code == "NoSuchKey" {
				// No thumbnails exists yet, fallback to full resolution
				fmt.Printf("No thumbnail found for '%v' falling back to full res\n", thumbPath)
				presignedURL, err = minioClient.PresignedGetObject(context.Background(), mediaBucket, imgPath, time.Second*1*60*60, reqParams)
				if err != nil {
					fmt.Println(err)
					return
				}

			} else {
				// A different error occured (e.g. access denied, bucket non-existant)
				log.Fatal(err)
			}

		} else {
			fmt.Println("Thumbnail exists:", objInfo)

			presignedURL, err = minioClient.PresignedGetObject(context.Background(), thumbnailBucket, thumbPath, time.Second*1*60*60, reqParams)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("getting thumb")
			fmt.Println(presignedURL)

		}

	} else {

		// Generates a presigned url which expires in a hour.
		presignedURL, err = minioClient.PresignedGetObject(context.Background(), mediaBucket, imgPath, time.Second*1*60*60, reqParams)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	// fmt.Println("Successfully generated presigned URL", presignedURL)
	c.Redirect(http.StatusSeeOther, presignedURL.String())
}

func listObjectsByPrefix(prefix string) []string {

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
			fmt.Println(object.Err)
			return ret
		}
		ret = append(ret, strings.TrimSuffix(object.Key, "/"))
	}
	return ret
}

func indexHandler(c *gin.Context) {

	tmpldata := struct {
		Title  string
		Albums []string
	}{
		Title:  "Albums",
		Albums: listObjectsByPrefix("/"),
	}

	c.HTML(http.StatusOK, "index.html", tmpldata)
}
