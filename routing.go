package gofakes3

import (
	"net/http"
	"strings"
)

// routeBase is a http.HandlerFunc that dispatches top level routes for
// GoFakeS3.
//
// URLs are assumed to break down into two common path segments, in the
// following format:
//   /<bucket>/<object>
//
// The operation for most of the core functionality is built around HTTP
// verbs, but outside the core functionality, the clean separation starts
// to degrade, especially around multipart uploads.
//
func (g *GoFakeS3) routeBase(w http.ResponseWriter, r *http.Request) {
	var (
		path   = strings.Trim(r.URL.Path, "/")
		parts  = strings.SplitN(path, "/", 2)
		bucket = parts[0]
		query  = r.URL.Query()
		object = ""
		err    error
	)

	if len(parts) == 2 {
		object = parts[1]
	}

	if uploadID := UploadID(query.Get("uploadId")); uploadID != "" {
		err = g.routeMultipartUpload(bucket, object, uploadID, w, r)

	} else if _, ok := query["uploads"]; ok {
		err = g.routeMultipartUploadBase(bucket, object, w, r)

	} else if bucket != "" && object != "" {
		err = g.routeObject(bucket, object, w, r)

	} else if bucket != "" {
		err = g.routeBucket(bucket, w, r)

	} else if r.Method == "GET" {
		err = g.getBuckets(w, r)

	} else {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		g.httpError(w, r, err)
	}
}

// routeObject oandles URLs that contain both a bucket path segment and an
// object path segment.
func (g *GoFakeS3) routeObject(bucket, object string, w http.ResponseWriter, r *http.Request) (err error) {
	switch r.Method {
	case "GET":
		return g.getObject(bucket, object, w, r)
	case "PUT":
		return g.createObject(bucket, object, w, r)
	case "DELETE":
		return g.deleteObject(bucket, object, w, r)
	case "HEAD":
		return g.headObject(bucket, object, w, r)
	default:
		return ErrMethodNotAllowed
	}
}

// routeBucket handles URLs that contain only a bucket path segment, not an
// object path segment.
func (g *GoFakeS3) routeBucket(bucket string, w http.ResponseWriter, r *http.Request) (err error) {
	switch r.Method {
	case "GET":
		return g.getBucket(bucket, w, r)
	case "PUT":
		return g.createBucket(bucket, w, r)
	case "DELETE":
		return g.deleteBucket(bucket, w, r)
	case "HEAD":
		return g.headBucket(bucket, w, r)
	case "POST":
		if _, ok := r.URL.Query()["delete"]; ok {
			return g.deleteMulti(bucket, w, r)
		} else {
			return g.createObjectBrowserUpload(bucket, w, r)
		}
	default:
		return ErrMethodNotAllowed
	}
}

// routeMultipartUploadBase operates on routes that contain '?uploads' in the
// query string. These routes may or may not have a value for bucket or object;
// this is validated and handled in the target handler functions.
func (g *GoFakeS3) routeMultipartUploadBase(bucket, object string, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		return g.listMultipartUploads(bucket, w, r)
	case "POST":
		return g.initiateMultipartUpload(bucket, object, w, r)
	default:
		return ErrMethodNotAllowed
	}
}

// routeMultipartUpload operates on routes that contain '?uploadId=<id>' in the
// query string.
func (g *GoFakeS3) routeMultipartUpload(bucket, object string, uploadID UploadID, w http.ResponseWriter, r *http.Request) error {
	switch r.Method {
	case "GET":
		return g.listMultipartUploadParts(bucket, object, uploadID, w, r)
	case "PUT":
		return g.putMultipartUploadPart(bucket, object, uploadID, w, r)
	case "DELETE":
		return g.abortMultipartUpload(bucket, object, uploadID, w, r)
	case "POST":
		return g.completeMultipartUpload(bucket, object, uploadID, w, r)
	default:
		return ErrMethodNotAllowed
	}
}