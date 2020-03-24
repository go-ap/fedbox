## Issues:
* Filtering is kinda broken. I need a better abstraction for it.
* loading outbox of an account shows non-public activities too: http://fedbox.git/actors/4b39b035-e38a-4f79-a3e0-14cc0798fe42/outbox
* ~~Replying doesn't thread correctly since we modified the logic away from `Context` being the thread starter.~~
* ~~natural language values are broken with latest go-ap/ActivityStreams commits.~~ Eg: see Name below: 
```json
{
    "id": "http://fedbox.git/objects/80e441fe-46bd-44be-aebe-19f782242213",
    "type": "Page",
    "name": {
        "": "",
        "-": "http://fedbox.git/objects/80e441fe-46bd-44be-aebe-19f782242213/shares"
    },
    "url": "https://tools.ietf.org/html/rfc5782"
}
```
* ~~**MAJOR Issue** quotes are escaped with extra backslashes when Marshaling~~
* ~~some GET tests on LikeNote don't seem to work~~ 
* ~~still doubly-escaping the `\n` and `\r` when encoding the JSON-Ld natural language value properties.~~

## Features:
* Make fedbox be usable as a package. Something similar to:
```go
chi.Route ("/", fedbox.Route())
// or
chi.Route("/activities", fedbox.Activities())
chi.Route("/objects", fedbox.Objects())
chi.Route("/actors", fedbox.Actors())

```
* ~~Undo activity (for Like/Dislike)~~
* ~~Fix OAuth2 logging in for users. Currently a valid username works with any pw.~~
* ~~item likes/dislikes counts in littr.go~~
