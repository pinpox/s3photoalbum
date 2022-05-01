package main

import (
	"context"
	"github.com/minio/minio-go/v7"
	"strings"
)

func listFirstObjectByPrefix(prefix string) (string, error) {

	log.Info("listing first in:", prefix)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// List objects
	objectCh := minioClient.ListObjects(ctx,config.S3MediaBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
		MaxKeys:   1,
	})

	ret := ""

	for object := range objectCh {
		if object.Err != nil {
			log.Error(object.Err)
			return "", object.Err
		}

		log.Info(object.Key)
		ret = strings.TrimPrefix(object.Key, prefix)
		log.Info("returening ", ret)
		break
	}
	return ret, nil
}

func listObjectsByPrefix(prefix string) ([]string, error) {

	log.Info("listing:", prefix)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// List objects
	objectCh := minioClient.ListObjects(ctx,config.S3MediaBucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	ret := []string{}
	for object := range objectCh {
		if object.Err != nil {
			log.Error(object.Err)
			return ret, object.Err
		}
		ret = append(ret, strings.TrimPrefix(strings.TrimSuffix(object.Key, "/"), prefix))
	}
	return ret, nil
}
