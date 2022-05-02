package main

import (
	"bytes"
	"context"
	"fmt"
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"s3photoalbum/internal"
)

var (
	minioClient *minio.Client
	config      s3photoalbum.ThumbnailerConfig
	log         *zap.SugaredLogger
)

func runCmd(cmd *exec.Cmd) (stdout, stderr string, err error) {

	var stdOut, stdErr bytes.Buffer
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr

	log.Debug(cmd.String())
	err = cmd.Run()
	if err != nil {
		log.Error(stdOut.String())
		log.Error(stdErr.String())
	}

	return stdOut.String(), stdErr.String(), err

}

func setExifOrientation(pathIn, orientation string) error {

	// shell ❯ exiftool -Orientation=6 -n test2.jpg
	//     1 image files updated

	cmdExiftool := exec.Command(
		config.ExifToolPath,
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
		config.ExifToolPath,
		"-s",
		"-s",
		"-s",
		"-Orientation",
		"-n",
		pathIn)

	stdOut, _, err := runCmd(cmdExiftool)
	log.Debug("Orientation string", stdOut)

	stdOut = strings.TrimSpace(stdOut)
	// Check for number in expected range
	orientN, err := strconv.ParseUint(stdOut, 10, 32)
	if err != nil || orientN > 10 {

		log.Error("ERROR parsing orientation", stdOut, err)
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

	// Try to get the exif orientation before conversion
	orientation, errExif := getExifOrientation(pathIn)

	cmdFfmpeg := exec.Command(
		config.FfmpegThumbnailerPath,
		"-i",
		pathIn,
		"-o",
		pathOut,
		"-s",
		config.ThumbnailSize,
	)

	stdOut, _, err := runCmd(cmdFfmpeg)
	if err != nil {
		log.Error("FFMpeg failed to extact thumbmail", stdOut)
		return err
	}

	log.Debug(stdOut)

	if errExif == nil {
		//ignore errors while setting orientation
		setExifOrientation(pathOut, orientation)
	}

	return err
}

func makeThumbnailByKey(key string) error {

	objInfo, err := minioClient.StatObject(context.Background(), config.S3MediaBucket, key, minio.StatObjectOptions{})
	if err != nil {
		log.Error(err)
		return err
	}

	return makeThumbnail(key, objInfo.ETag)
}

func makeThumbnail(key, etag string) (err error) {

	log.Debug("Making thumbnail for:", key, "etag:", etag)
	tmpInFileName := etag + path.Ext(key)
	tmpOutFileName := etag + path.Ext(key) + ".jpg"
	newKey := key + ".jpg"

	err = minioClient.FGetObject(
		context.Background(),
		config.S3MediaBucket,
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
		log.Error("Failed to retrieve original media:", key)

		return err
	}

	err = getThumbJPEG(tmpInFileName, tmpOutFileName)
	if err != nil {
		log.Error("Failed to extrat JPEG for:", key)
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
		config.S3ThumbnailBucket,
		newKey,
		tmpOutFileName,
		minio.PutObjectOptions{ContentType: "image/jpeg"},
	); err == nil {
		log.Info("Successfully uploaded bytes: ", info)
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

	thumbsCh := minioClient.ListObjects(ctx, config.S3ThumbnailBucket, minio.ListObjectsOptions{Recursive: true})
	mediaCh := minioClient.ListObjects(ctx, config.S3MediaBucket, minio.ListObjectsOptions{Recursive: true})

	var mediaKeys []string
	var thumbKeys []string

	for object := range mediaCh {
		if object.Err != nil {
			log.Error(object.Err)
			break
		}
		mediaKeys = append(mediaKeys, object.Key)
	}

	for object := range thumbsCh {
		if object.Err != nil {
			log.Error(object.Err)
			break
		}
		thumbKeys = append(thumbKeys, strings.TrimSuffix(object.Key, ".jpg"))
	}

	return difference(mediaKeys, thumbKeys)

}

func main() {

	config = s3photoalbum.LoadThumbnailerConfig()
	log = s3photoalbum.NewLogger(config.ModeDevelop)

	useSSL := true

	var err error

	// Initialize minio client object.
	minioClient, err = minio.New(config.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(config.S3AccessKey, config.S3SecretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		panic(err)
	}

	log.Info("Checking for missing thumbnails")
	missingThumbs := getMissingThumbnails()
	log.Info(len(missingThumbs), "thumbnails missing")

	for _, v := range missingThumbs {
		log.Info("Creating thumbnail for: %s\n", v)
		if err := makeThumbnailByKey(v); err != nil {
			log.Error("Error making thumbnail for: ", v)
		}
	}

	// Listen for bucket notifications
	for notificationInfo := range minioClient.ListenBucketNotification(context.Background(), config.S3MediaBucket, "", "", []string{
		"s3:ObjectCreated:*",
		// "s3:ObjectAccessed:*",
		// "s3:ObjectRemoved:*",
	}) {
		if notificationInfo.Err != nil {
			log.Error(notificationInfo.Err)
		}

		for _, k := range notificationInfo.Records {

			if !checkBucketKeyExists(k.S3.Object.Key+".jpg", config.S3ThumbnailBucket) {

				// No thumbnails exists yet, generate and upload
				if err = makeThumbnail(k.S3.Object.Key, k.S3.Object.ETag); err != nil {
					// Something happened while generating or uploading the thumbnail
					log.Error(err)
					continue
				}
			}
		}
	}
}

func checkBucketKeyExists(key, bucket string) bool {
	_, err := minioClient.StatObject(context.Background(), bucket, key, minio.StatObjectOptions{})

	if err != nil && minio.ToErrorResponse(err).Code != "NoSuchKey" {
		log.Error(err)
	}
	return err == nil
}
