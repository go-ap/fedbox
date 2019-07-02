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
insert into actors ("key", "iri", "type", "raw") values ('%s', '%s', 'Service', '%s'::jsonb);
-- name: insert-collection
insert into collections ("iri", "type") values ('%s', '%s');
-- name: add-to-collection
update collections set count = 1, elements = array_append(elements, '%s') WHERE iri = '%s';
-- name: drop-database
drop database "%s";
-- name: drop-role
drop role "%s";
-- name: truncate-tables
truncate table collections;
truncate table actors;
truncate table activities;
truncate table objects;
