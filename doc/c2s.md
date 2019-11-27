# Fed::BOX as an ActivityPub server supporting C2S interactions

Here I will do a dump of my assumptions regarding how the "client to server[1]" (or C2S) interactions should work on FedBOX.

The first one is that the C2S API will be structured as a REST(ful) API in respect to addressing objects. 

This means that every object on the server will have an unique URL that can be addressed at. 

This type of URL is called an "Internationalized Resource Identifier[2]" (or IRI) in the ActivityPub spec. 

It also represents the Object's ID in respect to the AP spec.

## API end-points

Fedbox has as a unique entry point for any non-authorized request. For convenience we'll assume that is the root path for the domain (eg: `https://federated.id/`)

We'll call this entry point the "Local Service's IRI", as it response consists of a Service Actor representing the current instance.

The Service, as an Actor, must have an Inbox collection, which we expose in the `https://federated.id/inbox` end-point.

It also represents the shared inbox for all actors created on the service.

## Collections

Since in the ActivityPub spec there is no schema for IRI generation for Objects, Activities and Actors and it's left as an implementation detail, we decided that for FedBOX we wanted to create three non-specified collection end-points, corresponding to each of these and serving as a base for every every entity's ID. 

These additional non-spec conforming collections are:

* https://federated.id/actors - where we can query all the actors on the instance.
* https://federated.id/activities - where we can query all the activities on the instance.
* https://federated.id/objects - where we can query the rest of the items.

## Object collections:

The object collections in the ActivitypPub spec are: `following`, `followers`, `liked`.
Additionally FedBOX has the previously mentioned `/actors` and `/objects` root end-points.

On these collections we can use the following filters:

  * *iri*: list of IRIs representing specific object ID's we want to load
  * *publishedDate*: [*] timestamp and operator for timestamp
  * *type*: list of Object types
  * *to*: list of IRIs
  * *cc*: list of IRIs
  * *audience*: list of IRIs
  * *generator*: list of IRIs, representing the Application actors that pushed the object
  * *url*: list of URLs

[*] Filter not yet implemented  

## Activity collections:

The activity collections in the ActivitypPub spec are: `outbox`, `inbox`, `likes`, `shares`, `replies`.
Additionally FedBOX supports the `/activities` root end-point.

In order to get the full representation of the items, after loading one of these collections, their Object properties need to be dereferenced and loaded again.

Besides the filters applicable to Object collections we have also:

  * *actor*: list of IRIs
  * *object* list of IRIs
  * *target*: list of IRIs

# The filtering

Filtering collections is done using query parameters corresponding to the lowercased value of the property's name it matches against.

All filters can be used multiple times in the URL. 

The matching logic is:

* Multiple values of same filter are matched by doing an union on the resulting sets.
* Different filters match by doing an intersection on the resulting sets of each filter.

We currently need to add the possibility of prepending operators to the values, so we can support negative matches, or some other types of filtering.

___

[1] See ActivityPub spec: https://www.w3.org/TR/activitypub/#client-to-server-interactions  
[2] See RFC3987: https://tools.ietf.org/html/rfc3987  
