# Fedbox

Here I will do a dump of my assumptions regarding how the "client to server[1]" (or C2S) interactions should work on FedBOX.

The first one is that the C2S API will be structured as a REST(ful) API in respect to addressing objects. 

This means that every object on the server will have an unique URL that can be addressed at. 

This type of URL is called an "Internationalized Resource Identifier[2]" (or IRI) in the ActivityPub spec. 

It also represents the Object's ID in respect to the AP spec.

## Entry points

Fedbox has as a unique entry point for any non-authorized request. For convenience we'll assume that is the root path for the domain (eg: `https://federated.id/`)

We'll call this entry point the "Local Service's IRI", as it response consists of a Service Actor representing the current instance.

The Service, as an Actor, must have an Inbox collection, which we expose in the `https://federated.id/inbox` end-point.

It also represents the shared inbox for all actors created on the service.

In the ActivityPub spec there is no schema for IRI generation for Objects, Activities and Actors. This is left as an implementation detail.

In our case we took the option of creating three non-specified collection end-points, corresponding to each of these. 

We have thus:

* https://federated.id/actors - where we can query all the actors on the instance.
* https://federated.id/activities - where we can query all the activities on the instance.
* https://federated.id/objects - where we can query the rest of the items.

## Object and Actor filtering

Ideally we want to support every property of an ActivityPub object as a filtering option. Eg: InReplyTo, can be an array of IRI values that the collection gets filtered against.

## Activities filtering

Additionally to the Object filtering properties, an activity can be filtered additionally by its "Object", "Actor" and "Target" fields.

___

[1] See ActivityPub spec: https://www.w3.org/TR/activitypub/#client-to-server-interactions  
[2] See RFC3987: https://tools.ietf.org/html/rfc3987  
