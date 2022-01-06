# s3flatten

[![Maintainability](https://api.codeclimate.com/v1/badges/9d4f7d8030936a8c0ce3/maintainability)](https://codeclimate.com/github/tkawachi/s3flatten/maintainability)

s3flatten copies a set of files in a nested directory on S3 to a flat directory.
In the destination directory, `/`, which is used as the delimiter of the directory,
is converted to `-` by default.

```
s3flatten --suffix=.txt s3://my-source/path1/ s3://my-destination/path2/

source path                      destination path
s3://my-source/path1/            s3://my-destination/path2/
    foo.txt            --COPY->         foo.txt
    bar.png
    abc/def.txt        --COPY->         abc-def.txt
    abc/def/ghi.txt    --COPY->         abc-def-ghi.txt
```

The source bucket and destination bucket can be the same.
If the source bucket and destination bucket are the same, the destination key prefix must
not be under the source key prefix.

## Usage

```
Usage: s3flatten [-hv] [-c value] [-d value] [-r value] [-s value] s3://src-bucket/path/to/src/ s3://dest-bucket/path/to/dest/
 -c, --concurrency=value
                    Number of goroutine for COPY operation [128]
 -d, --delimiter=value
                    Delimiter to replace '/' with to flatten path. [-]
 -h, --help         display help
 -r, --retry=value  Maximum number of retry for S3 operation [3]
 -s, --suffix=value
                    Copy only objects which has this suffix in key
 -v, --verbose      verbose output
```
