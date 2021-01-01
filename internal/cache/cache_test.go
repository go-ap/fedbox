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
		name string
		r    store
		args args
		want bool
	}{
		{
			name: "simple",
			r: store{
				c: iriMap{pub.IRI("example1"): &pub.Object{ID: pub.IRI("example1")}},
			},
			args: args{pub.IRI("example1")},
			want: true,
		},
		{
			name: "same_url",
			r: store{
				c: iriMap{ pub.IRI("http://example.com"): &pub.Actor{ID: pub.IRI("http://example.com")}},
			},
			args: args{pub.IRI("http://example.com")},
			want: true,
		},
		{
			name: "different_urls",
			r: store{
				c: iriMap{pub.IRI("http://example.com/inbox"): &pub.Actor{ID: pub.IRI("http://example.com")}},
			},
			args: args{pub.IRI("http://example.com")},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.Remove(tt.args.iri); got != tt.want {
				t.Errorf("Remove() = %v, want %v", got, tt.want)
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
