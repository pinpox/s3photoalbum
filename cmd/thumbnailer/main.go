package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	// "github.com/disintegration/imaging"
)

var minioClient *minio.Client
var mediaBucket string
var thumbnailBucket string
var thumbnailSize string
var ffmpegThumbnailerPath string

func getThumbJPEG(pathIn, pathOut string) error {

	// Usage: ffmpegthumbnailer [options]

	// Options:
	//   -i<s>   : input file
	//   -o<s>   : output file
	//   -s<n>   : thumbnail size (use 0 for original size) (default: 128)
	//   -t<n|s> : time to seek to (percentage or absolute time hh:mm:ss) (default: 10%)
	//   -q<n>   : image quality (0 = bad, 10 = best) (default: 8)
	//   -c      : override image format (jpeg, png or rgb) (default: determined by filename)
	//   -a      : ignore aspect ratio and generate square thumbnail
	//   -f      : create a movie strip overlay
	//   -m      : prefer embedded image metadata over video content
	//   -w      : workaround issues in old versions of ffmpeg
	//   -v      : print version number
	//   -h      : display this help

	var err error

	size, err := strconv.ParseUint(thumbnailSize, 10, 32)
	if err != nil || size < 5 {
		thumbnailSize = "256"
	}

	cmd := exec.Command(
		ffmpegThumbnailerPath,
		"-i",
		pathIn,
		"-o",
		pathOut,
		"-s",
		thumbnailSize,
	)

	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Println(stderr.String())
		return err
	}

	cmd.Wait()
	return err
}

func makeThumbnail(key, contentType, etag string) (err error) {

	fmt.Println("Making thumbnail for:", key, "etag:", etag)
	tmpInFileName := etag + path.Ext(key)
	tmpOutFileName := etag + path.Ext(key) + ".jpg"
	newKey := key + ".jpg"

	err = minioClient.FGetObject(
		context.Background(),
		mediaBucket,
		key,
		tmpInFileName,
		minio.GetObjectOptions{},
	)

	defer func() {
		if err = os.Remove(tmpInFileName); err != nil {
			return
		}
	}()

	if err != nil {
		return err
	}

	err = getThumbJPEG(tmpInFileName, tmpOutFileName)
	if err != nil {
		return err
	}

	// Make sure thumbnail file is deleted
	defer func() {
		if err = os.Remove(tmpOutFileName); err != nil {
			return
		}
	}()

	if info, err := minioClient.FPutObject(
		context.Background(),
		thumbnailBucket,
		newKey,
		tmpOutFileName,
		minio.PutObjectOptions{ContentType: contentType},
	); err == nil {
		fmt.Println("Successfully uploaded bytes: ", info)
	}

	return err
}

// difference returns the elements in `a` that aren't in `b`.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

func getMissingThumbnails() []string {

	ctx, cancel := context.WithCancel(context.Background())

	defer cancel()

	thumbsCh := minioClient.ListObjects(ctx, thumbnailBucket, minio.ListObjectsOptions{Recursive: true})
	mediaCh := minioClient.ListObjects(ctx, mediaBucket, minio.ListObjectsOptions{Recursive: true})

	var mediaKeys []string
	var thumbKeys []string

	for object := range mediaCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			break
		}
		mediaKeys = append(mediaKeys, object.Key)
	}

	for object := range thumbsCh {
		if object.Err != nil {
			fmt.Println(object.Err)
			break
		}
		thumbKeys = append(thumbKeys, strings.TrimSuffix(object.Key, ".jpg"))
	}

	return difference(mediaKeys, thumbKeys)

}

func main() {

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKeyID := os.Getenv("S3_ACCESSKEY")
	secretAccessKey := os.Getenv("S3_SECRETKEY")
	mediaBucket = os.Getenv("S3_BUCKET_MEDIA")
	thumbnailBucket = os.Getenv("S3_BUCKET_THUMBNAILS")
	thumbnailSize = os.Getenv("THUMBNAIL_SIZE")
	ffmpegThumbnailerPath = os.Getenv("FFMPEGTHUMBNAILER_PATH")

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

	fmt.Println("Missing thumbnails")
	fmt.Println(getMissingThumbnails())

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
					if err = makeThumbnail(k.S3.Object.Key, k.S3.Object.ContentType, k.S3.Object.ETag); err != nil {
						// Something happened while generating or uploading the thumbnail
						fmt.Println(err)
						continue
					}

				} else {
					// A different error occured (e.g. access denied, bucket non-existant)
					log.Fatal(err)
				}

			} else {
				fmt.Println("Thumbnail exists:", objInfo.Key)
			}
		}
	}
}
