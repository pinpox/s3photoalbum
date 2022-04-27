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
var exiftoolPath string

func runCmd(cmd *exec.Cmd) (stdout, stderr string, err error) {

	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	fmt.Println(cmd.String())
	err = cmd.Run()
	if err != nil {
		fmt.Println(stdOut.String())
		fmt.Println(stdErr.String())
	}

	return stdOut.String(), stdErr.String(), err

}

func setExifOrientation(pathIn, orientation string) error {

	// shell ❯ exiftool -Orientation=6 -n test2.jpg
	//     1 image files updated

	cmdExiftool := exec.Command(
		exiftoolPath,
		"-Orientation="+orientation,
		"-n",
		"-overwrite_original",
		pathIn)

	_, _, err := runCmd(cmdExiftool)

	return err

}

func getExifOrientation(pathIn string) (string, error) {

	// shell ❯ exiftool -s -s -s -Orientation -n testdata/wrong_rotate.jpg
	// 6

	cmdExiftool := exec.Command(
		exiftoolPath,
		"-s",
		"-s",
		"-s",
		"-Orientation",
		"-n",
		pathIn)

	stdOut, _, err := runCmd(cmdExiftool)
	fmt.Println("Orientation string", stdOut)

	stdOut = strings.TrimSpace(stdOut)
	// Check for number in expected range
	orientN, err := strconv.ParseUint(stdOut, 10, 32)
	if err != nil || orientN > 10 {

		fmt.Println("ERROR parsing orientation", stdOut, err)
		return "", err
	}

	return fmt.Sprint(orientN), err

}

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

	// Try to get the exif orientation before conversion
	orientation, errExif := getExifOrientation(pathIn)

	cmdFfmpeg := exec.Command(
		ffmpegThumbnailerPath,
		"-i",
		pathIn,
		"-o",
		pathOut,
		"-s",
		thumbnailSize,
	)

	stdOut, _, err := runCmd(cmdFfmpeg)
	if err != nil {
		fmt.Println("FFMpeg failed to extact thumbmail", stdOut)
		return err
	}

	fmt.Println(stdOut)

	if errExif == nil {
		//ignore errors while setting orientation
		setExifOrientation(pathOut, orientation)
	}

	return err
}

func makeThumbnailByKey(key string) error {

	objInfo, err := minioClient.StatObject(context.Background(), mediaBucket, key, minio.StatObjectOptions{})
	if err != nil {
		fmt.Println(err)
		return err
	}

	return makeThumbnail(key, objInfo.ETag)
}

func makeThumbnail(key, etag string) (err error) {

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
		fmt.Println("Failed to retrieve original media:", key)

		return err
	}

	err = getThumbJPEG(tmpInFileName, tmpOutFileName)
	if err != nil {
		fmt.Println("Failed to extrat JPEG for:", key)
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
		minio.PutObjectOptions{ContentType: "image/jpeg"},
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
	exiftoolPath = os.Getenv("EXIFTOOL_PATH")

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

	fmt.Println("Checking for missing thumbnails")
	missingThumbs := getMissingThumbnails()
	fmt.Println(len(missingThumbs), "thumbnails missing")

	for _, v := range missingThumbs {
		fmt.Printf("Creating thumbnail for: %s\n", v)
		if err := makeThumbnailByKey(v); err != nil {
			fmt.Println("Error making thumbnail for: ", v)
		}
	}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(context.Background(), mediaBucket, "", "", []string{
		"s3:ObjectCreated:*",
		// "s3:ObjectAccessed:*",
		// "s3:ObjectRemoved:*",
	}) {
		if notificationInfo.Err != nil {
			fmt.Println(notificationInfo.Err)
		}

		for _, k := range notificationInfo.Records {

			if !checkBucketKeyExists(k.S3.Object.Key+".jpg", thumbnailSize) {

				// No thumbnails exists yet, generate and upload
				if err = makeThumbnail(k.S3.Object.Key, k.S3.Object.ETag); err != nil {
					// Something happened while generating or uploading the thumbnail
					fmt.Println(err)
					continue
				}
			}
		}
	}
}

func checkBucketKeyExists(key, bucket string) bool {
	_, err := minioClient.StatObject(context.Background(), bucket, key, minio.StatObjectOptions{})

	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
		fmt.Println("Error: ", err)
	}
	return err == nil
}
