## Bugs:
* loading outbox of an account shows non-public activities too: http://fedbox.git/actors/4b39b035-e38a-4f79-a3e0-14cc0798fe42/outbox
* ~~some GET tests on LikeNote don't seem to work~~ 
* still doubly-escaping the `\n` and `\r` when encoding the JSON-Ld natural language value properties.

## Features:
* Make fedbox be usable as a package. Something similar to:
```go
chi.Route ("/", fedbox.Route())
// or
chi.Route("/activities", fedbox.Activities())
chi.Route("/objects", fedbox.Objects())
chi.Route("/actors", fedbox.Actors())

```
* Undo activity
* item likes/dislikes counts
