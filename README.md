# Xray

Deep learning based object detection for video.

## Install

```sh
go get -d -u github.com/minio/xray
cd $GOPATH/src/github.com/minio/xray
make
```

## Run

```sh
xray
INFO[0000] Started listening on ws://192.168.1.106:8080
INFO[0000] Started listening on ws://127.0.0.1:8080
```

## Advanced

To build/update cascade assets, install [`go-bindata-assetfs`](https://github.com/elazarl/go-bindata-assetfs).

```sh
cd $GOPATH/src/github.com/minio/xray/cascade
go-bindata-assetfs -pkg cascade haar_face_0.xml lbp_face.xml
```
