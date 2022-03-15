package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"sync"

	"github.com/target/goalert/util/log"
)

type etagHandler struct {
	tags   map[string]string
	h      http.Handler
	fs     http.FileSystem
	mx     sync.Mutex
	static bool
}

func NewEtagFileServer(files http.FileSystem, static bool) http.Handler {
	return &etagHandler{
		tags:   make(map[string]string),
		h:      http.FileServer(files),
		fs:     files,
		static: static,
	}
}

func (e *etagHandler) etag(name string) string {
	e.mx.Lock()
	defer e.mx.Unlock()

	if tag, ok := e.tags[name]; e.static && ok {
		return tag
	}

	f, err := e.fs.Open(name)
	if err != nil {
		log.Log(context.Background(), err)
		e.tags[name] = ""
		return ""
	}
	defer f.Close()

	h := sha256.New()

	_, err = io.Copy(h, f)
	if err != nil {
		log.Log(context.Background(), err)
		e.tags[name] = ""
		return ""
	}

	tag := `W/"` + hex.EncodeToString(h.Sum(nil)) + `"`
	e.tags[name] = tag
	return tag
}

func (e *etagHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if tag := e.etag(req.URL.Path); tag != "" {
		w.Header().Set("Cache-Control", "public; max-age=60, stale-while-revalidate=600, stale-if-error=259200")
		w.Header().Set("ETag", tag)
	}

	e.h.ServeHTTP(w, req)
}
