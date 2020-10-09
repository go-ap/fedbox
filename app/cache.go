package app

import (
	"crypto/md5"
	"fmt"
	pub "github.com/go-ap/activitypub"
	"net/http"
	"path"
	"strings"
)

type reqCache map[pub.IRI]pub.Item

func cacheKey(r *http.Request) pub.IRI {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if auth := r.Header.Get("Authorization"); len(auth) != 0 {
		if strings.Contains(auth, "Bearer") {
			tok := strings.TrimPrefix(auth, "Bearer ")
			host = fmt.Sprintf("%s@%s", tok, host)
		}
		if strings.Contains(auth, "Signature") {
			// TODO(marius): this is untested
			fields := strings.Split(auth, " ")
			if len(fields) > 1 {
				sigField := fields[1]
				host = fmt.Sprintf("%02x@%s", md5.Sum([]byte(sigField)), host)
			}
		}
	}
	return pub.IRI(fmt.Sprintf("%s://%s", scheme, path.Join(host, r.RequestURI)))
}

func (r reqCache) has(iri pub.IRI) bool {
	_, ok := r[iri]
	return ok
}

func (r reqCache) get(iri pub.IRI) pub.Item {
	if it, ok := r[iri]; ok {
		return it
	}
	return nil
}

func (r *reqCache) set(iri pub.IRI, it pub.Item) {
	(*r)[iri] = it
}

func (r *reqCache) remove(iri pub.IRI) bool {
	if iri == pub.PublicNS {
		return false
	}
	for key := range *r {
		// TODO(marius): I need to play around with this a bit
		if key.Contains(iri, false) {
			delete(*r, key)
		}
	}
	return true
}
