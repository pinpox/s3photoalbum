package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client
var mediaBucket string
var thumbnailBucket string

var albumTemplate *template.Template
var indexTemplate *template.Template

func main() {
	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	mediaBucket = os.Getenv("S3_BUCKET_MEDIA")
	thumbnailBucket = os.Getenv("S3_BUCKET_THUMBNAILS")

	useSSL := true

	var err error

	albumTemplate, err = template.New("album.html").Funcs(template.FuncMap{
		"incolumn": func(colNum, index int) bool { return index%4 == colNum },
	}).ParseFiles("templates/album.html")
	if err != nil {
		log.Fatalln(err)
	}

	indexTemplate, err = template.New("index.html").ParseFiles("templates/index.html")
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

	fmt.Println("starting")

	router := httprouter.New()
	router.GET("/", indexHandler)
	router.GET("/albums/:album", albumHandler)
	router.GET("/albums/:album/:image", imageHandler)
	router.ServeFiles("/static/*filepath", http.Dir("static"))
	log.Fatal(http.ListenAndServe(":8080", router))

}

func albumHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	ad := struct {
		Title  string
		Images []string
	}{
		Title:  ps.ByName("album"),
		Images: listObjectsByPrefix(ps.ByName("album") + "/"),
	}

	err := albumTemplate.Execute(w, ad)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func imageHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	res := r.URL.Query().Get("thumbnail")
	thumbnail, err := strconv.ParseBool(res)
	if err != nil {
		thumbnail = false
	}

	imgPath := ps.ByName("album") + "/" + ps.ByName("image")

	// Set request parameters for content-disposition.
	reqParams := make(url.Values)
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

	http.Redirect(w, r, presignedURL.String(), http.StatusSeeOther)

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

func indexHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	tmpldata := struct {
		Title  string
		Albums []string
	}{
		Title:  "Albums",
		Albums: listObjectsByPrefix("/"),
	}

	err := indexTemplate.Execute(w, tmpldata)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
