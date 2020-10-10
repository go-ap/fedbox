package app

import (
	pub "github.com/go-ap/activitypub"
	"github.com/go-ap/fedbox/activitypub"
	"reflect"
	"testing"
)

func Test_reqCache_get(t *testing.T) {
	type args struct {
		iri pub.IRI
	}
	tests := []struct {
		name string
		r    reqCache
		args args
		want pub.Item
	}{
		{
			name: "",
			r: reqCache{},
			args: args{},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.get(tt.args.iri); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("get() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_reqCache_has(t *testing.T) {
	type args struct {
		iri pub.IRI
	}
	tests := []struct {
		name string
		r    reqCache
		args args
		want bool
	}{
		{
			name: "simple",
			r: reqCache{
				pub.IRI("example1"): &pub.Object{ID: pub.IRI("example1")},
			},
			args: args{pub.IRI("example1")},
			want: true,
		},
		{
			name: "same_url",
			r: reqCache{
				pub.IRI("http://example.com"): &pub.Actor{ID: pub.IRI("http://example.com")},
			},
			args: args{pub.IRI("http://example.com")},
			want: true,
		},
		{
			name: "different_urls",
			r: reqCache{
				pub.IRI("http://example.com/inbox"): &pub.Actor{ID: pub.IRI("http://example.com")},
			},
			args: args{pub.IRI("http://example.com")},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.has(tt.args.iri); got != tt.want {
				t.Errorf("has() = %v, want %v", got, tt.want)
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
		r    reqCache
		args args
		want bool
	}{
		{
			name: "simple",
			r: reqCache{
				pub.IRI("example1"): &pub.Object{ID: pub.IRI("example1")},
			},
			args: args{pub.IRI("example1")},
			want: true,
		},
		{
			name: "same_url",
			r: reqCache{
				pub.IRI("http://example.com"): &pub.Actor{ID: pub.IRI("http://example.com")},
			},
			args: args{pub.IRI("http://example.com")},
			want: true,
		},
		{
			name: "different_urls",
			r: reqCache{
				pub.IRI("http://example.com/inbox"): &pub.Actor{ID: pub.IRI("http://example.com")},
			},
			args: args{pub.IRI("http://example.com")},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.r.remove(tt.args.iri); got != tt.want {
				t.Errorf("remove() = %v, want %v", got, tt.want)
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
		r    reqCache
		args args
	}{
		{
			name: "",
			r: reqCache{},
			args: args{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}

func Test_cacheKey(t *testing.T) {
	type args struct {
		f *activitypub.Filters
	}
	tests := []struct {
		name string
		args args
		want pub.IRI
	}{
		{
			name: "example.com",
			args: args{f: &activitypub.Filters{IRI: "http://example.com"}},
			want: pub.IRI("http://example.com"),
		},
		{
			name: "authenticated",
			args: args{f: &activitypub.Filters{IRI: "http://example.com", Authenticated: &pub.Actor{ID:"http://example.com/jdoe"}}},
			want: pub.IRI("http://jdoe@example.com"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cacheKey(tt.args.f); got != tt.want {
				t.Errorf("cacheKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
