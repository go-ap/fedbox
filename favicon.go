package fedbox

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/go-ap/errors"
	"github.com/go-ap/fedbox/internal/assets"
	"github.com/nfnt/resize"
	"github.com/sergeymakinen/go-ico"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func svgDecode(in io.Reader) (image.Image, error) {
	icon, err := oksvg.ReadIconStream(in)
	if err != nil {
		return nil, err
	}

	w := int(icon.ViewBox.W)
	h := int(icon.ViewBox.H)

	icon.SetTarget(0, 0, icon.ViewBox.W, icon.ViewBox.H)

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	icon.Draw(rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, rgba, rgba.Bounds())), 1)

	return rgba, nil
}

const maxFaviconSize = 192

var objectCacheDuration = 168 * time.Hour // 7 days

func ServeFavIcon() http.HandlerFunc {
	logo, err := assets.Assets.Open("logo.svg")
	if err != nil {
		return errors.HandleError(errors.NewNotFound(err, "nothing here")).ServeHTTP
	}
	defer logo.Close()

	orig, err := svgDecode(logo)
	if err != nil {
		return errors.HandleError(errors.NotFoundf("failed to open image: %s", err)).ServeHTTP
	}

	if m := orig.Bounds().Max; m.X > maxFaviconSize || m.Y > maxFaviconSize {
		var sw uint = maxFaviconSize
		var sh uint = 0
		if m.X < m.Y {
			// NOTE(marius): if the height is larger than the width, we use that as the main resize axis
			sw = 0
			sh = maxFaviconSize
		}
		orig = resize.Resize(sw, sh, orig, resize.MitchellNetravali)
	}

	raw := make([]byte, 0)
	buf := bytes.NewBuffer(raw)
	if err = ico.Encode(buf, orig); err != nil {
		return errors.HandleError(errors.NotFoundf("failed to create favicon: %s", err)).ServeHTTP
	}

	raw = buf.Bytes()
	eTag := fmt.Sprintf(`"%2x"`, md5.Sum(raw))

	var updatedAt time.Time
	if fi, err := logo.Stat(); err == nil {
		updatedAt = fi.ModTime()
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mime.TypeByExtension(".ico"))
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(raw)))
		w.Header().Set("Vary", "Accept")
		w.Header().Set("ETag", eTag)
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(objectCacheDuration.Seconds())))

		status := http.StatusOK
		uaHasItem := requestMatchesETag(r.Header, eTag) || requestMatchesLastModified(r.Header, updatedAt)
		if uaHasItem {
			status = http.StatusNotModified
		}
		if !updatedAt.IsZero() {
			w.Header().Set("Last-Modified", updatedAt.Format(time.RFC1123))
		}

		w.WriteHeader(status)
		if r.Method == http.MethodGet && !uaHasItem {
			_, _ = w.Write(raw)
		}
	}
}

func requestMatchesLastModified(h http.Header, updated time.Time) bool {
	modifiedSince := h.Get("If-Modified-Since")
	modSinceTime, err := time.Parse(time.RFC1123, modifiedSince)
	if err != nil {
		return false
	}
	return modSinceTime.Equal(updated) || modSinceTime.After(updated)
}

func requestMatchesETag(h http.Header, eTag string) bool {
	noneMatchValues, ok := h["If-None-Match"]
	if !ok {
		return false
	}

	for _, ifNoneMatch := range noneMatchValues {
		if ifNoneMatch == eTag {
			return true
		}
	}
	return false
}
