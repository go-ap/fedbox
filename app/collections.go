package app

import (
	as "github.com/go-ap/activitystreams"
	"github.com/go-ap/fedbox/internal/errors"
	j "github.com/go-ap/jsonld"
	"net/http"
	"strings"
)

func renderCollection(c as.CollectionInterface) ([]byte, error) {
	return j.Marshal(c)
}

// CollectionHandlerFn is the type that we're using to represent handlers that return ActivityStreams
// Collection objects. It needs to implement the http.Handler interface
type CollectionHandlerFn func(http.ResponseWriter, *http.Request) (as.CollectionInterface, error)

// ValidMethod validates if the current handler can process the current request
func (c CollectionHandlerFn) ValidMethod( r *http.Request) bool {
	return r.Method != http.MethodGet && r.Method != http.MethodHead
}

// ServeHTTP implements the http.Handler interface for the CollectionHandlerFn type
func (c CollectionHandlerFn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var dat []byte
	var status int

	if c.ValidMethod(r) {
		status = http.StatusNotAcceptable
		dat, _ = errors.Render(errors.MethodNotAllowedf("invalid HTTP method"))
	}

	if col, err := c(w, r); err != nil {
		// HandleError
		status = http.StatusInternalServerError
		dat, _ = errors.Render(err)
	} else {
		if dat, err = renderCollection(col); err != nil {
			status = http.StatusInternalServerError
			dat, _ = errors.Render(err)
		} else {
			status = http.StatusOK
		}
	}

	w.WriteHeader(status)
	if r.Method == http.MethodGet {
		w.Write(dat)
	}
}

// CollectionType
type CollectionType string

const (
	Unknown   = CollectionType("")
	Outbox    = CollectionType("outbox")
	Inbox     = CollectionType("inbox")
	Shares    = CollectionType("shares")
	Replies   = CollectionType("replies") // activitystreams
	Following = CollectionType("following")
	Followers = CollectionType("followers")
	Liked     = CollectionType("liked")
	Likes     = CollectionType("likes")
)

// CollectionTyper allows external packages to tell us which collection the current HTTP request addresses
type CollectionTyper interface {
	Type(r *http.Request) CollectionType
}

type DefaultCollectionTyper struct{}

func (d DefaultCollectionTyper) Type(r *http.Request) CollectionType {
	if r.URL == nil || len(r.URL.Path) == 0 {
		return Unknown
	}
	var col string
	pathElements := strings.Split(r.URL.Path[1:], "/") // Skip first /
	for i := len(pathElements) - 1; i >= 0; i-- {
		col = pathElements[i]
		if typ := getValidActivityCollection(col); typ != Unknown {
			return typ
		}
		if typ := getValidObjectCollection(col); typ != Unknown {
			return typ
		}
	}

	return CollectionType(col)
}

var validActivityCollection = []CollectionType{
	Outbox,
	Inbox,
	Likes,
	Shares,
	Replies, // activitystreams
}

func getValidActivityCollection(typ string) CollectionType {
	for _, t := range validActivityCollection {
		if strings.ToLower(typ) == string(t) {
			return t
		}
	}
	return Unknown
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Activities
func ValidActivityCollection(typ string) bool {
	return getValidActivityCollection(typ) != Unknown
}

var validObjectCollection = []CollectionType{
	Following,
	Followers,
	Liked,
}

func getValidObjectCollection(typ string) CollectionType {
	for _, t := range validObjectCollection {
		if strings.ToLower(typ) == string(t) {
			return t
		}
	}
	return Unknown
}

// ValidActivityCollection shows if the current ActivityPub end-point type is a valid one for handling Objects
func ValidObjectCollection(typ string) bool {
	return getValidObjectCollection(typ) != Unknown
}

func getValidCollection(typ string) CollectionType {
	if typ := getValidActivityCollection(typ); typ != Unknown {
		return typ
	}
	if typ := getValidObjectCollection(typ); typ != Unknown {
		return typ
	}
	return Unknown
}

func ValidCollection(typ string) bool {
	return getValidCollection(typ) != Unknown
}

// HandleActivityCollection serves content from the outbox, inbox, likes, shares and replies end-points
// that return ActivityPub collections containing activities
func HandleActivityCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	// TODO(marius): move typer instantiation outside the handler, so we can pass it from outside
	typer := DefaultCollectionTyper{}
	collection :=  typer.Type(r)
	return nil, errors.NotImplementedf("%s", collection)
}

// HandleObjectCollection serves content from following, followers, liked, and likes end-points
// that return ActivityPub collections containing plain objects
func HandleObjectCollection(w http.ResponseWriter, r *http.Request) (as.CollectionInterface, error) {
	// TODO(marius): move typer instantiation outside the handler, so we can pass it from outside
	typer := DefaultCollectionTyper{}
	collection :=  typer.Type(r)

	return nil, errors.NotImplementedf("%s", collection)
}
