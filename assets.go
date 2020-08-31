//+build !dev
//go:generate go run -tags $(ENV) aletheia.icu/broccoli -src ./templates -var assets -o internal/assets/assets.gen.go

package assets
