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
    "created_at" timetz default current_timestamp,
    "type" types,
    "raw" jsonb
);
-- name: create-activitypub-activities
create table activities
(
    "id"  serial not null constraint activities_pkey primary key,
    "key" varchar constraint activities_key_key unique,
    "iri" varchar constraint activities_iri_key unique,
    "created_at" timetz default current_timestamp,
    "type" types,
    "raw" jsonb
);
-- name: create-activitypub-actors
create table actors
(
    "id"  serial not null constraint actors_pkey primary key,
    "key" varchar constraint actors_key_key unique,
    "iri" varchar constraint actors_iri_key unique,
    "created_at" timetz default current_timestamp,
    "type" types,
    "raw" jsonb
);
