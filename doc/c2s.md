# Fedbox

Here I will do a dump of my assumptions regarding how the "client to server" (from here on, referred to as C2S[1]) interactions should work between littr.me as a client and it's ActivityPub server, fedbox.

* The C2S API will be structured as close as possible to a REST API in respect to addressing objects. Ie, every object on the server will have an unique URL that can be found at.

This type of URL is called an IRI (Internationalized Resource Identifier, see RFC3987[2]) in the ActivityPub spec.

## Entry points

Fedbox has as a unique entry point for any non-authorized requests. For convenience we'll assume that is the root path for the domain (eg: https://federated.id/)

We'll call this entry point the "Service's IRI", as it consists of a Service Actor representing the current instance.

The Service, as an Actor, must have an Inbox collection, which we expose in the https://federated.id/inbox end-point.
It also represents the shared inbox for all actors created on the service.

*Problem*: in the ActivityPub spec there is no schema for IRI addressing for Objects, Activities and Actors. This is an implementation detail.
In the case of fedbox we took the option of creating three collection end-points, corresponding to each of these. 

We have thus:

* https://federated.id/objects 
* https://federated.id/activities
* https://federated.id/actors 

## Object and Actor filtering

Ideally we want to support every property of an ActivityPub object as a filtering option. Eg: InReplyTo, can be an array of IRI values that the collection gets filtered against.

## Activities filtering

Additionally to the Object filtering properties, an activity can be filtered additionally by its "Object", "Actor" and "Target" fields.

___

[1] https://www.w3.org/TR/activitypub/#client-to-server-interactions  
[2] https://tools.ietf.org/html/rfc3987  
