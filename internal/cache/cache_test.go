package cache

import (
	"reflect"
	"sync"
	"testing"

	vocab "github.com/go-ap/activitypub"
)

func Test_reqCache_get(t *testing.T) {
	type args struct {
		iri vocab.IRI
	}
	tests := []struct {
		name string
		r    store
		args args
		want vocab.Item
	}{
		{
			name: "",
			r:    store{},
			args: args{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Get(tt.args.iri); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() = %v, want %v", got, tt.want)
			}
		})
	}
}

type iriItemMap = map[vocab.IRI]vocab.Item

func iriMap(objects ...iriItemMap) sync.Map {
	s := sync.Map{}
	for _, entry := range objects {
		for k, v := range entry {
			s.Store(k, v)
		}
	}
	return s
}

func Test_reqCache_remove(t *testing.T) {
	type args struct {
		iri vocab.IRI
	}
	tests := []struct {
		name      string
		r         store
		args      args
		want      bool
		leftovers vocab.IRIs
	}{
		{
			name: "simple",
			r: store{
				enabled: true,
				c:       iriMap(iriItemMap{"example1": &vocab.Object{ID: "example1"}}),
			},
			args:      args{vocab.IRI("example1")},
			want:      true,
			leftovers: vocab.IRIs{},
		},
		{
			name: "same_url",
			r: store{
				enabled: true,
				c:       iriMap(iriItemMap{"http://example.com": &vocab.Actor{ID: "http://example.com"}}),
			},
			args:      args{vocab.IRI("http://example.com")},
			want:      true,
			leftovers: vocab.IRIs{},
		},
		{
			name: "different_urls",
			r: store{
				enabled: true,
				c:       iriMap(iriItemMap{"http://example.com/inbox": &vocab.Actor{ID: "http://example.com"}}),
			},
			args:      args{vocab.IRI("http://example.com")},
			want:      true,
			leftovers: vocab.IRIs{},
		},
		{
			name: "with_replies",
			r: store{
				enabled: true,
				c: iriMap(iriItemMap{
					"http://example.com/elefant": vocab.IRI("http://example.com/elefant"),
					"http://example.com/test": &vocab.Object{
						ID:      "http://example.com/test",
						Replies: vocab.IRI("http://example.com/test/replies"),
					},
					"http://example.com/test/replies": vocab.ItemCollection{
						vocab.IRI("http://example.com/0"),
						vocab.IRI("http://example.com/1"),
					},
				}),
			},
			args:      args{vocab.IRI("http://example.com/test")},
			want:      true,
			leftovers: vocab.IRIs{vocab.IRI("http://example.com/elefant")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Remove(tt.args.iri); got != tt.want {
				t.Errorf("Remove() = %v, want %v", got, tt.want)
			}
			for _, iri := range tt.leftovers {
				v, ok := tt.r.c.Load(iri)
				if v == nil || !ok {
					t.Errorf("IRI should be in cache, but not found  %s", iri)
				}
				tt.r.c.Delete(iri)
			}
		})
	}
}

func Test_reqCache_set(t *testing.T) {
	type args struct {
		iri vocab.IRI
		it  vocab.Item
	}
	tests := []struct {
		name string
		r    store
		args args
	}{
		{
			name: "",
			r:    store{},
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
