package cache

import (
	pub "github.com/go-ap/activitypub"
	"reflect"
	"testing"
)

func Test_reqCache_get(t *testing.T) {
	type args struct {
		iri pub.IRI
	}
	tests := []struct {
		name string
		r    store
		args args
		want pub.Item
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

func Test_reqCache_remove(t *testing.T) {
	type args struct {
		iri pub.IRI
	}
	tests := []struct {
		name      string
		r         store
		args      args
		want      bool
		leftovers pub.IRIs
	}{
		{
			name: "simple",
			r: store{
				enabled: true,
				c: iriMap{pub.IRI("example1"): &pub.Object{ID: pub.IRI("example1")}},
			},
			args:      args{pub.IRI("example1") },
			want:      true,
			leftovers: pub.IRIs{},
		},
		{
			name: "same_url",
			r: store{
				enabled: true,
				c: iriMap{pub.IRI("http://example.com"): &pub.Actor{ID: pub.IRI("http://example.com")}},
			},
			args:      args{pub.IRI("http://example.com")},
			want:      true,
			leftovers: pub.IRIs{},
		},
		{
			name: "different_urls",
			r: store{
				enabled: true,
				c: iriMap{pub.IRI("http://example.com/inbox"): &pub.Actor{ID: pub.IRI("http://example.com")}},
			},
			args:      args{pub.IRI("http://example.com")},
			want:      true,
			leftovers: pub.IRIs{},
		},
		{
			name: "with_replies",
			r: store{
				enabled: true,
				c: iriMap{
					pub.IRI("http://example.com/elefant"): pub.IRI("http://example.com/elefant"),
					pub.IRI("http://example.com/test"): &pub.Object{
						ID: pub.IRI("http://example.com/test"),
						Replies: pub.IRI("http://example.com/test/replies"),
					},
					pub.IRI("http://example.com/test/replies"): pub.ItemCollection{
						pub.IRI("http://example.com/0"),
						pub.IRI("http://example.com/1"),
					},
				},
			},
			args:      args{pub.IRI("http://example.com/test")},
			want:      true,
			leftovers: pub.IRIs{pub.IRI("http://example.com/elefant")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Remove(tt.args.iri); got != tt.want {
				t.Errorf("Remove() = %v, want %v", got, tt.want)
			}
			if len(tt.leftovers) != len(tt.r.c) {
				t.Errorf("Cache length missmatch %d, want %d", len(tt.r.c), len(tt.leftovers))
			}
			for _, iri := range tt.leftovers {
				if tt.r.c[iri] == nil {
					t.Errorf("IRI should be in cache, but not found  %s", iri)
				} else {
					delete(tt.r.c, iri)
				}
			}
			if len(tt.r.c) > 0 {
				t.Errorf("IRIs should not be in cache, but still found  %#v", tt.r.c)
			}
		})
	}
}

func Test_reqCache_set(t *testing.T) {
	type args struct {
		iri pub.IRI
		it  pub.Item
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
