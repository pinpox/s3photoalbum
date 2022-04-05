package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/disintegration/imaging"
)

var minioClient *minio.Client
var mediaBucket string
var thumbnailBucket string

func makeThumbnail(key, contentType string) error {

	fmt.Println("Making thumbnail for:", key)

	// Get source media and create an image from it
	object, err := minioClient.GetObject(context.Background(), mediaBucket, key, minio.GetObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return nil
	}

	var img image.Image
	img, err = imaging.Decode(object, imaging.AutoOrientation(true))

	if err != nil {
		return err
	}

	// For fixed size thumbnails
	// thumbnail := imaging.Thumbnail(img, 100, 100, imaging.CatmullRom)
	// Leaving height at 0 keeps the original aspect ratio
	thumbnail := imaging.Resize(img, 200, 0, imaging.CatmullRom)

	var b bytes.Buffer
	w := bufio.NewWriter(&b)

	err = jpeg.Encode(w, thumbnail, &jpeg.Options{Quality: 90})

	if err != nil {
		fmt.Println(err)
		return err
	}

	reader := bytes.NewReader(b.Bytes())

	newFileName := key + ".jpg"
	fmt.Println(newFileName)

	uploadInfo, err := minioClient.PutObject(context.Background(), thumbnailBucket, newFileName, reader, int64(len(b.Bytes())), minio.PutObjectOptions{ContentType: contentType})
	if err != nil {
		fmt.Println(err)
		return err
	}

	fmt.Println("Successfully uploaded bytes: ", uploadInfo)

	return nil
}

func main() {

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	mediaBucket = os.Getenv("S3_BUCKET_MEDIA")
	thumbnailBucket = os.Getenv("S3_BUCKET_THUMBNAILS")

	useSSL := true

	var err error

	// Initialize minio client object.
	minioClient, err = minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(context.Background(), mediaBucket, "", "", []string{
		"s3:ObjectCreated:*",
		"s3:ObjectAccessed:*",
		"s3:ObjectRemoved:*",
	}) {
		if notificationInfo.Err != nil {
			fmt.Println(notificationInfo.Err)
		}

		for _, k := range notificationInfo.Records {

			// Check if object exists. If it does not an error will be thrown.
			objInfo, err := minioClient.StatObject(context.Background(), thumbnailBucket, k.S3.Object.Key+".jpg", minio.StatObjectOptions{})
			if err != nil {

				errResponse := minio.ToErrorResponse(err)
				if errResponse.Code == "NoSuchKey" {
					// No thumbnails exists yet, generate and upload
					if err = makeThumbnail(k.S3.Object.Key, k.S3.Object.ContentType); err != nil {
						// Something happened while generating or uploading the thumbnail
						fmt.Println(err)
						continue
					}

				} else {
					// A different error occured (e.g. access denied, bucket non-existant)
					log.Fatal(err)
				}

			} else {
				fmt.Println("Thumbnail exists:", objInfo)
			}
		}
	}
}
