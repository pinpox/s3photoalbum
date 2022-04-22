package main

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

func albumHandler(c *gin.Context) {

	td := templateData{
		Context: c,
		Data: struct {
			Album   string
			Images  []string
			Context *gin.Context
		}{
			Album:   c.Param("album"),
			Images:  listObjectsByPrefix(path.Join(c.GetString("username"), c.Param("album")) + "/"),
			Context: c,
		},
	}

	c.HTML(http.StatusOK, "album.html", td)
}

func imageHandler(c *gin.Context) {

	res := c.DefaultQuery("thumbnail", "false")
	thumbnail, err := strconv.ParseBool(res)
	if err != nil {
		thumbnail = false
	}

	imgPath := path.Join(c.GetString("username"), c.Param("album"), c.Param("image"))

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
				log.Errorf("No thumbnail found for '%v' falling back to full res\n", thumbPath)
				presignedURL, err = minioClient.PresignedGetObject(context.Background(), mediaBucket, imgPath, time.Second*1*60*60, reqParams)
				if err != nil {
					log.Error(err)
					return
				}

			} else {
				// A different error occured (e.g. access denied, bucket non-existant)
				log.Fatal(err)
			}

		} else {
			log.Debug("Thumbnail exists:", objInfo)

			presignedURL, err = minioClient.PresignedGetObject(context.Background(), thumbnailBucket, thumbPath, time.Second*1*60*60, reqParams)
			if err != nil {
				log.Error(err)
				return
			}
			log.Debug(presignedURL)

		}

	} else {

		// Generates a presigned url which expires in a hour.
		presignedURL, err = minioClient.PresignedGetObject(context.Background(), mediaBucket, imgPath, time.Second*1*60*60, reqParams)
		if err != nil {
			log.Error(err)
			return
		}
	}
	//log.Infof("Successfully generated presigned URL", presignedURL)
	c.Redirect(http.StatusSeeOther, presignedURL.String())
}
