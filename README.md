# s3photoalbum

<p align="center">
  <img height="200" src="./logo.svg">
</p>

Show photoalbums from S3-compatible buckets, e.g. Minio.

The application consists of a server that serves the images as gallery and a
thumbnailer that can be run seprately. Both the photos and the thumbnails will
are saved in S3 buckets.

## Requirements

Two S3 buckets.

> TODO: bucket permissions exmaple

## Configuration

The server and tumbnailer are both configured via environment varibles. All
varibles without a default must be set.

### Common settings

| Variable              | Default | Description                                                 |
|-----------------------|---------|-------------------------------------------------------------|
| `S3_ENDPOINT`         |         | S3 Endpoint without scheme                                  |
| `S3_ACCESS_KEY`       |         | S3 Access key                                               |
| `S3_SECRET_KEY`       |         | S3 Secret key                                               |
| `S3_MEDIA_BUCKET`     |         | Bucket where the media files are stored                     |
| `S3_THUMBNAIL_BUCKET` |         | Bucket to place the Thumbnails in                           |
| `S3_USE_SSL`          | `true`  | Whether to use SSL (https://) to connect to the endpoint    |
| `MODE_DEVELOP`        | `false` | Run in development mode (verbose logging)                   |
| `RESOURCES_DIR`       | `.`     | Directory containing `/templates` and `/static` directories |

Different access and secret keys can be specified for the server and the
tumbnailer. While the server will need only read access to both buckets, the
thumbnailer needs to be able to write to the thumbnails bucket. The bucket
containing the media files may be read-only in both cases.

### Server-specific settings

| Variable         | Default     | Description                                                    |
|------------------|-------------|----------------------------------------------------------------|
| `JWT_KEY`        |             | Key to use for JWT authentication (`openssl rand -base64 172`) |
| `INITIAL_USER`   | `admin`     | Initial user to create                                         |
| `INITIAL_PASS`   | `admin`     | Plain-text password for intial user                            |
| `HOST`           | `localhost` | Hostname of the application                                    |
| `LISTEN_ADDRESS` | `127.0.0.1` | Address to listen on                                           |
| `LISTEN_PORT`    | `7788`      | Port to listen on                                              |

Don't forget to change the intial password after intial setup!

### Thumbnailer-specific settings

Both `ffmpegthumbnailer` and `exiftool` are used to generate the thumbnails. The
can be installed on most linux distributions via the package manager.

| Variable                  | Default | Description                                                                       |
|---------------------------|---------|-----------------------------------------------------------------------------------|
| `THUMBNAIL_SIZE`          | `300"`  | Size of generated thumbnails (in pixels)                                          |
| `FFMPEG_THUMBNAILER_PATH` |         | Path containing [ffmpegthumbnailer](https://github.com/dirkvdb/ffmpegthumbnailer) |
| `EXIF_TOOL_PATH`          |         | Path containing [exiftool](https://exiftool.org/)                                 |

## Run

Start the thumbnailer and the server separately with the above variables set


## s3fs mount bucket
```
export AWS_ACCESS_KEY_ID="XXXXXXXXXXXXXXXXXXXX"
export AWS_SECRET_ACCESS_KEY="YYYYYYYYYYYYYYYYYYY"
s3fs bucket-name ~/s3mount -o 'use_path_request_style,url=https://s3.my.host'
```
