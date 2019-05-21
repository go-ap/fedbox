-- name: create-role-with-pass
CREATE ROLE "%s" LOGIN PASSWORD '%s';
-- name: create-db-for-role
CREATE DATABASE "%s" OWNER "%s";

-- name: create-activitypub-types-enum
CREATE TYPE "types" AS ENUM (
    'Object',
    'Link',
    'Activity',
    'IntransitiveActivity',
    'Actor',
    'Collection',
    'OrderedCollection',
    'CollectionPage',
    'OrderedCollectionPage',
    'Article',
    'Audio',
    'Document',
    'Event',
    'Image',
    'Note',
    'Page',
    'Place',
    'Profile',
    'Relationship',
    'Tombstone',
    'Video',
    'Mention',
    'Application',
    'Group',
    'Organization',
    'Person',
    'Service',
    'Accept',
    'Add',
    'Announce',
    'Arrive',
    'Block',
    'Create',
    'Delete',
    'Dislike',
    'Flag',
    'Follow',
    'Ignore',
    'Invite',
    'Join',
    'Leave',
    'Like',
    'Listen',
    'Move',
    'Offer',
    'Question',
    'Reject',
    'Read',
    'Remove',
    'TentativeReject',
    'TentativeAccept',
    'Travel',
    'Undo',
    'Update',
    'View'
    );
-- name: create-activitypub-objects
create table objects
(
    "id"  serial not null constraint objects_pkey primary key,
    "key" varchar constraint objects_key_key unique,
    "iri" varchar constraint objects_iri_key unique,
    "created_at" timestamptz default current_timestamp,
    "type" types,
    "raw" jsonb
);
-- name: create-activitypub-activities
create table activities
(
    "id"  serial not null constraint activities_pkey primary key,
    "key" varchar constraint activities_key_key unique,
    "iri" varchar constraint activities_iri_key unique,
    "created_at" timestamptz default current_timestamp,
    "updated_at" timestamptz default NULL,
    "type" types,
    "raw" jsonb
);
-- name: create-activitypub-actors
create table actors
(
    "id"  serial not null constraint actors_pkey primary key,
    "key" varchar constraint actors_key_key unique,
    "iri" varchar constraint actors_iri_key unique,
    "created_at" timestamptz default current_timestamp,
    "updated_at" timestamptz default NULL,
    "type" types,
    "raw" jsonb
);
-- name: create-activitypub-collections
create table collections (
     "id"  serial not null constraint collections_pkey primary key,
     "iri" varchar constraint collections_iri_key unique,
     "type" types,
     "created_at" timestamptz default current_timestamp,
     "updated_at" timestamptz default NULL,
     "count" int DEFAULT 0,
     "elements" varchar[] default NULL
);
-- name: insert-service-actor
insert into actors ("key", "iri", "type", "raw")
values ('d3ab037c-0f15-4c09-b635-3d6e201c11aa', 'https://federated.id/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa', 'Service', '{"@context": ["https://www.w3.org/ns/activitystreams"],"id": "https://federated.id/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa","type": "Service","name": "self","inbox": "https://federated.id/inbox", "audience": [
  "https://www.w3.org/ns/activitystreams#Public"]}');
-- name: insert-activities-collection
insert into collections ("iri", "type") values ('http://fedbox.git:4000/activities', 'OrderedCollection');
-- name: insert-actors-collection
insert into collections ("iri", "type", "count") values ('http://fedbox.git:4000/actors', 'OrderedCollection', 1, 'https://federated.id/actors/d3ab037c-0f15-4c09-b635-3d6e201c11aa');
-- name: insert-objects-collection
insert into collections ("iri", "type") values ('http://fedbox.git:4000/objects', 'OrderedCollection');
