# s3photoalbum

Show photoalbums from S3-compatible buckets, e.g. Minio.

The application consists of a server that serves the images as gallery and a
thumbnailer that can be run seprately. Both the photos and the thumbnails will
are saved in S3 buckets.

## Requirements

Two S3 buckets.

## Configuration

The applicatoin is configured via environment variables. The server and the
thumbnailer may use separate credentials. Only the thumnailer needs write access
to the thumbnails bucket and only read access is needed on the media bucket.

```
export S3_ENDPOINT="s3.myhost.com"
export S3_ACCESSKEY="XXXXXXXXXXXXXXXX"
export S3_SECRETKEY="YYYYYYYYYYYYYYYYYYYYYYYYYY"
export S3_BUCKET_MEDIA="photos-bucket"
export S3_BUCKET_THUMBNAILS="thumnails-bucket"
export S3_SSL=true

# openssl rand -base64 172
export JWT_KEY="XXXXXXX"

export INITIAL_USER=admin
export INITIAL_PASS=admin
```

## Run

Start the thumbnailer and the server separately with the above variables set



