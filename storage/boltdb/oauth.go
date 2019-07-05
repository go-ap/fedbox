package boltdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/go-ap/errors"
	"github.com/openshift/osin"
	"github.com/sirupsen/logrus"
	"time"
)

// Storage implements interface "github.com/RangelReale/osin".Storage and interface "github.com/ory/osin-storage".Storage
type Storage struct {
	d       *bolt.DB
	root    []byte
	logFn   loggerFn
	errFn   loggerFn
}

// New returns a new postgres storage instance.
func NewOAuthStore(c Config) *Storage {
	d, _ := bolt.Open(c.Path, 0600, nil)
	return &Storage{
		d:     d,
		root:  []byte(c.BucketName),
		logFn: c.LogFn,
		errFn: c.ErrFn,
	}
}

// Clone the storage if needed. For example, using mgo, you can clone the session with session.Clone
// to avoid concurrent access problems.
// This is to avoid cloning the connection at each method access.
// Can return itself if not a problem.
func (s *Storage) Clone() osin.Storage {
	return s
}

// Close the resources the Storage potentially holds (using Clone for example)
func (s *Storage) Close() {
	//s.db.Close()
}

type cl struct {
	Id          string
	Secret      string
	RedirectUri string
	Extra       json.RawMessage
}

const clientsBucket = "clients"

// GetClient loads the client by id
func (s *Storage) GetClient(id string) (osin.Client, error) {
	c := osin.DefaultClient{}
	err := s.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cl := cl{}
		cb := rb.Bucket([]byte(clientsBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, clientsBucket)
		}
		raw := cb.Get([]byte(id))
		if err := json.Unmarshal(raw, &cl); err != nil {
			return err
		}
		c.Id = cl.Id
		c.Secret = cl.Secret
		c.RedirectUri = cl.RedirectUri
		c.UserData = cl.Extra
		return nil
	})

	return &c, err
}

// UpdateClient updates the client (identified by it's id) and replaces the values with the values of client.
func (s *Storage) UpdateClient(c osin.Client) error {
	data, err := assertToString(c.GetUserData())
	if err != nil {
		s.errFn(logrus.Fields{"id": c.GetId()}, err.Error())
		return err
	}
	cl := cl {
		Id: c.GetId(),
		Secret: c.GetSecret(),
		RedirectUri: c.GetRedirectUri(),
		Extra: json.RawMessage(data),
	}
	raw, err := json.Marshal(cl)
	if err != nil {
		return err
	}
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(clientsBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, clientsBucket)
		}
		return cb.Put([]byte(cl.Id), raw)
	})
}

// CreateClient stores the client in the database and returns an error, if something went wrong.
func (s *Storage) CreateClient(c osin.Client) error {
	return s.UpdateClient(c)
}

// RemoveClient removes a client (identified by id) from the database. Returns an error if something went wrong.
func (s *Storage) RemoveClient(id string) (err error) {
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(clientsBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, clientsBucket)
		}
		return cb.Delete([]byte(id))
	})
}

// SaveAuthorize saves authorize data.
func (s *Storage) SaveAuthorize(data *osin.AuthorizeData) (err error) {
	extra, err := assertToString(data.UserData)
	if err != nil {
		s.errFn(logrus.Fields{"id": data.Client.GetId(), "code": data.Code}, err.Error())
		return err
	}

	auth := auth{
		data.Client.GetId(),
		data.Code,
		time.Duration(data.ExpiresIn),
		data.Scope,
		data.RedirectUri,
		data.State,
		data.CreatedAt,
		json.RawMessage(extra),
	}
	raw, err := json.Marshal(auth)
	if err != nil {
		return err
	}
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(authorizeBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, authorizeBucket)
		}
		return cb.Put([]byte(data.Code), raw)
	})
}

type auth struct {
	Client      string
	Code        string
	ExpiresIn   time.Duration
	Scope       string
	RedirectURI string
	State       string
	CreatedAt   time.Time
	Extra       json.RawMessage
}

var authorizeBucket = "authorize"

// LoadAuthorize looks up AuthorizeData by a code.
// Client information MUST be loaded together.
// Optionally can return error if expired.
func (s *Storage) LoadAuthorize(code string) (*osin.AuthorizeData, error) {
	var data osin.AuthorizeData

	err := s.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		auth := auth{}
		cb := rb.Bucket([]byte(authorizeBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, clientsBucket)
		}
		raw := cb.Get([]byte(code))
		if err := json.Unmarshal(raw, &auth); err != nil {
			return err
		}
		data.Code = auth.Code
		data.ExpiresIn = int32(auth.ExpiresIn)
		data.Scope = auth.Scope
		data.RedirectUri = auth.RedirectURI
		data.State = auth.State
		data.CreatedAt = auth.CreatedAt
		data.UserData = auth.Extra

		c, err := s.GetClient(auth.Client)
		if err != nil {
			return err
		}

		if data.ExpireAt().Before(time.Now()) {
			s.errFn(logrus.Fields{"code": code}, err.Error())
			return errors.Errorf("Token expired at %s.", data.ExpireAt().String())
		}

		data.Client = c
		return nil
	})

	return &data, err
}

// RemoveAuthorize revokes or deletes the authorization code.
func (s *Storage) RemoveAuthorize(code string) (err error) {
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(authorizeBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, authorizeBucket)
		}
		return cb.Delete([]byte(code))
	})
}

// SaveAccess writes AccessData.
// If RefreshToken is not blank, it must save in a way that can be loaded using LoadRefresh.
func (s *Storage) SaveAccess(data *osin.AccessData) (err error) {
	prev := ""
	authorizeData := &osin.AuthorizeData{}

	if data.AccessData != nil {
		prev = data.AccessData.AccessToken
	}

	if data.AuthorizeData != nil {
		authorizeData = data.AuthorizeData
	}

	extra, err := assertToString(data.UserData)
	if err != nil {
		s.errFn(logrus.Fields{"id": data.Client.GetId()}, err.Error())
		return err
	}

	if data.RefreshToken != "" {
		if err := s.saveRefresh(data.RefreshToken, data.AccessToken); err != nil {
			s.errFn(logrus.Fields{"id": data.Client.GetId()}, err.Error())
			return err
		}
	}

	if data.Client == nil {
		return errors.Newf("data.Client must not be nil")
	}

	acc := acc{
		Client:       data.Client.GetId(),
		Authorize:    authorizeData.Code,
		Previous:     prev,
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		ExpiresIn:    time.Duration(data.ExpiresIn),
		Scope:         data.Scope,
		RedirectURI:  data.RedirectUri,
		CreatedAt:    data.CreatedAt,
		Extra:        json.RawMessage(extra),
	}
	raw, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(accessBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, accessBucket)
		}
		return cb.Put([]byte(authorizeData.Code), raw)
	})
}

type acc struct {
	Client       string
	Authorize    string
	Previous     string
	AccessToken  string
	RefreshToken string
	ExpiresIn    time.Duration
	Scope        string
	RedirectURI  string
	CreatedAt    time.Time
	Extra        json.RawMessage
}

const accessBucket = "access"

// LoadAccess retrieves access data by token. Client information MUST be loaded together.
// AuthorizeData and AccessData DON'T NEED to be loaded if not easily available.
// Optionally can return error if expired.
func (s *Storage) LoadAccess(code string) (*osin.AccessData, error) {
	var result osin.AccessData

	err := s.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		var acc acc
		cb := rb.Bucket([]byte(accessBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, accessBucket)
		}
		raw := cb.Get([]byte(code))
		if err := json.Unmarshal(raw, &acc); err != nil {
			return err
		}
		result.AccessToken = acc.AccessToken
		result.RefreshToken = acc.RefreshToken
		result.ExpiresIn = int32(acc.ExpiresIn)
		result.Scope = acc.Scope
		result.RedirectUri = acc.RedirectURI
		result.CreatedAt = acc.CreatedAt
		result.UserData = acc.Extra
		client, err := s.GetClient(acc.Client)
		if err != nil {
			s.errFn(logrus.Fields{"code": code, "table": "access", "operation": "select",}, err.Error())
			return err
		}

		result.Client = client
		result.AuthorizeData, _ = s.LoadAuthorize(acc.Authorize)
		prevAccess, _ := s.LoadAccess(acc.Previous)
		result.AccessData = prevAccess
		return nil
	})

	return &result, err
}

// RemoveAccess revokes or deletes an AccessData.
func (s *Storage) RemoveAccess(code string) (err error) {
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(accessBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, accessBucket)
		}
		return cb.Delete([]byte(code))
	})
}

type ref struct {
	Access string
}

const refreshBucket = "refresh"

// LoadRefresh retrieves refresh AccessData. Client information MUST be loaded together.
// AuthorizeData and AccessData DON'T NEED to be loaded if not easily available.
// Optionally can return error if expired.
func (s *Storage) LoadRefresh(code string) (*osin.AccessData, error) {
	var ref ref
	err := s.d.View(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(refreshBucket))
		prefix := []byte(code)
		u := cb.Cursor()
		if u == nil {
			return errors.Errorf("Invalid bucket cursor %s/%s", s.root, refreshBucket)
		}
		for k, v := u.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = u.Next() {
			if err := json.Unmarshal(v, &ref); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.LoadAccess(ref.Access)
}

// RemoveRefresh revokes or deletes refresh AccessData.
func (s *Storage) RemoveRefresh(code string) error {
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(refreshBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, refreshBucket)
		}
		return cb.Delete([]byte(code))
	})
}

func (s *Storage) saveRefresh(refresh, access string) (err error) {
	ref := ref{
		Access: access,
	}
	raw, err := json.Marshal(ref)
	if err != nil {
		return err
	}
	return  s.d.Update(func(tx *bolt.Tx) error {
		rb := tx.Bucket(s.root)
		if rb == nil {
			return errors.Errorf("Invalid bucket %s", s.root)
		}
		cb := rb.Bucket([]byte(refreshBucket))
		if cb == nil {
			return errors.Newf("Invalid bucket %s/%s", s.root, refreshBucket)
		}
		return cb.Put([]byte(refresh), raw)
	})
}

func assertToString(in interface{}) (string, error) {
	var ok bool
	var data string
	if in == nil {
		return "", nil
	} else if data, ok = in.(string); ok {
		return data, nil
	} else if byt, ok := in.([]byte); ok {
		return string(byt), nil
	} else if byt, ok := in.(json.RawMessage); ok {
		return string(byt), nil
	} else if str, ok := in.(fmt.Stringer); ok {
		return str.String(), nil
	}
	return "", errors.Errorf(`Could not assert "%v" to string`, in)
}
