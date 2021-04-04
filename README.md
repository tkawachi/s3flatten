# s3flatten

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

## Usage

```
Usage: s3flatten [-hv] [-c value] [-d value] [-s value] s3://src-bucket/path/to/src/ s3://dest-bucket/path/to/dest/
 -c, --concurrency=value
                Number of goroutine for COPY operation [128]
 -d, --delimiter=value
                Delimiter to replace '/' with to flatten path. [-]
 -h, --help     display help
 -s, --suffix=value
                Copy only objects which has this suffix in key
 -v, --verbose  verbose output
```

## Notes

If you get the following error, specify the region of the source bucket in `AWS_REGION` environment variable.

> 2021/04/04 10:20:51 operation error S3: ListObjectsV2, https response error StatusCode: 301, RequestID: 00000000000000, HostID: buQzm/exF9GqDm+xkQLy2ID1gmyXO4rvAC4pez4387c/KopGjV8pt/iBbdUAnAABWZKC8fjX2qg=, api error PermanentRedirect: The bucket you are attempting to access must be addressed using the specified endpoint. Please send all future requests to this endpoint.
