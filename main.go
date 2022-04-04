package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/julienschmidt/httprouter"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var photoBucket string

var albumTemplate *template.Template

func main() {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	photoBucket = os.Getenv("S3_BUCKET")

	useSSL := true

	var err error

	albumTemplate, err = template.New("layout.html").Funcs(template.FuncMap{
		"incolumn": func(colNum, index int) bool { return index%4 == colNum },
	}).ParseFiles("layout.html")
	if err != nil {
		log.Fatalln(err)
	}

	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/albums/:album", albumHandler)
	router.GET("/albums/:album/:image", imageHandler)
	router.ServeFiles("/static/*filepath", http.Dir("static"))
	log.Fatal(http.ListenAndServe(":8080", router))

}

func albumHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	objectCh := minioClient.ListObjects(ctx, photoBucket, minio.ListObjectsOptions{
		Prefix:    ps.ByName("album") + "/",
		Recursive: false,
	})

	ad := albumData{
		Title:  ps.ByName("album"),
		Images: []string{},
	}

	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		ad.Images = append(ad.Images, object.Key)
	}

	err := albumTemplate.Execute(w, ad)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type albumData struct {
	Title  string
	Images []string
}

func imageHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	imgPath := ps.ByName("album") + "/" + ps.ByName("image")

	object, err := minioClient.GetObject(context.Background(), photoBucket, imgPath, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return
	}

	buf, err := ioutil.ReadAll(object)

	if err != nil {
		fmt.Println(err)
		return
	}

	w.Write(buf)

}

func indexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	// List objects
	objectCh := minioClient.ListObjects(ctx, photoBucket, minio.ListObjectsOptions{
		Prefix:    "/",
		Recursive: false,
	})

	for object := range objectCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			return
		}
		fmt.Println(object.Key)
		fmt.Fprintf(w, "- %s\n", object.Key)

	}
}
