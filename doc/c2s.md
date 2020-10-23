# Fed::BOX as an ActivityPub server supporting C2S interactions

Here I will do a dump of my assumptions regarding how the "client to server[1]" (or C2S) interactions should work on FedBOX.

The first one is that the C2S API will be structured as a REST(ful) API in respect to addressing objects. 

This means that every object on the server will have an unique URL that can be addressed at. 

This type of URL is called an "Internationalized Resource Identifier[2]" (or IRI) in the ActivityPub spec. 

It also represents the Object's ID in respect to the AP spec.

## API end-points

FedBOX has as a unique entry point for any non-authorized request. For convenience we'll assume that is the root path for the domain (eg: `https://federated.id/`)

We'll call this entry point the "Local Service's IRI", as it response consists of a Service Actor representing the current instance.

The Service, as an Actor, must have an Inbox collection, which we expose in the `https://federated.id/inbox` end-point.

It also represents the shared inbox for all actors created on the service.

## Collections

Since in the ActivityPub spec there is no schema for IRI generation for Objects, Activities and Actors and it's left as an implementation detail, we decided that for FedBOX we wanted to create three non-specified collection end-points, corresponding to each of these and serving as a base for every every entity's ID. 

These additional non-spec conforming collections are:

* https://federated.id/actors - where we can query all the actors on the instance.
* https://federated.id/activities - where we can query all the activities on the instance.
* https://federated.id/objects - where we can query the rest of the object types.

## Object collections:

An object collection, represents any collection that contains only ActivityPub Objects.
The object collections in the ActivitypPub spec are: `following`, `followers`, `liked`.
Additionally FedBOX has the previously mentioned `/actors` and `/objects` root end-points.

On these collections we can use the following filters:

  * **iri**: list of IRIs representing specific object ID's we want to load
  * **type**: list of Object types
  * **to**: list of IRIs
  * **cc**: list of IRIs
  * **audience**: list of IRIs
  * **url**: list of URLs

## Activity collections:

An activity collection, represents any collection that contains only ActivityPub Activities.
The activity collections in the ActivitypPub spec are: `outbox`, `inbox`, `likes`, `shares`, `replies`.
Additionally FedBOX supports the `/activities` root end-point.

In order to get the full representation of the items, after loading one of these collections, their Object properties need to be dereferenced and loaded again.

Besides the filters applicable to Object collections we have also:

  * **actor**: list of IRIs
  * **object** list of IRIs
  * **target**: list of IRIs

# The filtering

Filtering collections is done using query parameters corresponding to the snakeCased value of the property's name it matches against.

For end-points that return collection of activities, filtering can be done on the activity's actor/object/target properties 
by composing the filter name with a matching prefix for the child property:

Eg:  
`object.url=https://example.fed`  
`object.iri=https://example.fed/objects/{uuid}`  

All filters can be used multiple times in the URL. 

The matching logic is:

* Multiple values of same filter are matched by doing a union on the resulting sets.
* Different filters keys match by doing an intersection on the resulting sets of each filter.

The filtering values support basic operators to be used. They are:

For string based properties:

* `key=~value` - match `value` as a substring 
* `key=!value` - match everything that's doesn't contain `value` as a substring

For date based properties

* `key=>value` - match everything that has `key` property after the `value`
* `key=<value` - match everything that has `key` property before the `value`
___

[1] See ActivityPub spec: https://www.w3.org/TR/activitypub/#client-to-server-interactions  
[2] See RFC3987: https://tools.ietf.org/html/rfc3987  
