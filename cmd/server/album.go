package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

type Album struct {
	gorm.Model
	Name  string `gorm:"not null"`
	Cover string `gorm:"not null"`
}

func getAlbumsByUsername(username string) ([]Album, error) {

	var albums []Album
	albumNames, err := listObjectsByPrefix(username + "/")

	for _, v := range albumNames {
		coverImg, err2 := listFirstObjectByPrefix(username + "/" + v + "/")
		if err2 != nil {
			return albums, err2
		}
		albums = append(albums, Album{
			Name:  v,
			Cover: "/albums/" + v + "/" + coverImg,
		})
	}
	return albums, err
}

func albumHandler(c *gin.Context) {

	images, err := listObjectsByPrefix(path.Join(c.GetString("username"), c.Param("album")) + "/")

	if err != nil {
		log.Error(err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	td := templateData{
		Context: c,
		Data: struct {
			Album   string
			Images  []string
			Context *gin.Context
		}{
			Album:   c.Param("album"),
			Images:  images,
			Context: c,
		},
	}

	c.HTML(http.StatusOK, "album.html", td)
}

func checkBucketKeyExists(key, bucket string) bool {
	_, err := minioClient.StatObject(context.Background(), bucket, key, minio.StatObjectOptions{})
	return err == nil
}

func getFullResURI(imgPath string) string {

	reqParams := make(url.Values)

	if !checkBucketKeyExists(imgPath, mediaBucket) {
		log.Warnf("Image %s does not exist", imgPath)
		return "/static/missing.png"
	}

	// Generates a presigned url which expires in a hour.
	presignedURL, err := minioClient.PresignedGetObject(context.Background(), mediaBucket, imgPath, time.Second*1*60*60, reqParams)
	if err != nil {
		log.Warn(err)
		return "/static/missing.png"
	}

	log.Debug("Found full-res URL: ", presignedURL.String())
	return presignedURL.String()
}

func getThumbnailURI(imgPath string) string {
	// Set request parameters for content-disposition.
	reqParams := make(url.Values)
	// TODO for download
	// reqParams.Set("response-content-disposition", "attachment; filename=\""+ps.ByName("image")+"\"")

	thumbPath := imgPath + ".jpg"

	// Check if the real file exists
	if !checkBucketKeyExists(imgPath, mediaBucket) {
		return "/static/missing.png"
	}

	// Check if a thumbnail exists
	if !checkBucketKeyExists(thumbPath, thumbnailBucket) {
		return "/static/missing.png"
	}

	presignedURL, err := minioClient.PresignedGetObject(context.Background(), thumbnailBucket, thumbPath, time.Second*1*60*60, reqParams)
	if err != nil {
		log.Error(err)
		return "/static/missing.png"
	}

	log.Debug("Found thumbnail URL: ", presignedURL.String())
	return presignedURL.String()

}

func imageHandler(c *gin.Context) {

	imgPath := path.Join(c.GetString("username"), c.Param("album"), c.Param("image"))

	// Ignore error if any and default to false
	thumbnail, _ := strconv.ParseBool(c.Query("thumbnail"))

	if thumbnail {
		c.Redirect(http.StatusSeeOther, getThumbnailURI(imgPath))
	} else {
		c.Redirect(http.StatusSeeOther, getFullResURI(imgPath))
	}
}
