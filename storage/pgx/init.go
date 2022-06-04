//go:build storage_pgx || storage_all || (!storage_boltdb && !storage_fs && !storage_badger && !storage_sqlite)

package pgx

const (
	dropDatabase       = ``
	dropRole           = ``
	createRoleWithPass = ``
	createDbForRole    = ``
	extensionPgcrypto  = `create extension if not exists pgcrypto with schema public; `

	extensionLtree = `create extension if not exists ltree with schema public; `

	dropTables = `
DROP TABLE IF EXISTS votes CASCADE;
DROP TABLE IF EXISTS items CASCADE;
DROP TABLE IF EXISTS accounts CASCADE;
DROP TABLE IF EXISTS instances CASCADE;
DROP TABLE IF EXISTS objects CASCADE;
DROP TABLE IF EXISTS activities CASCADE;
DROP TABLE IF EXISTS actors CASCADE;
`

	truncateTables = `
TRUNCATE votes RESTART IDENTITY CASCADE;
TRUNCATE accounts RESTART IDENTITY CASCADE;
TRUNCATE items RESTART IDENTITY CASCADE;
TRUNCATE instances RESTART IDENTITY CASCADE;
TRUNCATE objects RESTART IDENTITY CASCADE;
TRUNCATE activities RESTART IDENTITY CASCADE;
TRUNCATE actors RESTART IDENTITY CASCADE;
`

	createAccounts = `
create table accounts (
  id serial constraint accounts_pk primary key,
  key char(32) unique,
  handle varchar,
  email varchar unique,
  score bigint default 0,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp,
  metadata jsonb default '{}',
  flags bit(8) default 0::bit(8)
); `

	createItems = `
create table items (
  id serial constraint items_pk primary key,
  key char(32) unique,
  mime_type varchar default NULL,
  title varchar default NULL,
  data text default NULL,
  score bigint default 0,
  path ltree default NULL,
  submitted_by int references accounts(id),
  submitted_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp,
  metadata jsonb default '{}',
  flags bit(8) default 0::bit(8)
); `

	createVotes = `
create table votes (
  id serial constraint votes_pk primary key,
  submitted_by int references accounts(id),
  submitted_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp,
  item_id  int references items(id),
  weight int,
  flags bit(8) default 0::bit(8),
  constraint unique_vote_submitted_item unique (submitted_by, item_id)
); `

	createInstances = `
create table instances
(
  id serial constraint instances_pk primary key,
  name varchar not null,
  description text,
  url varchar unique not null,
  inbox varchar unique,
  metadata jsonb default '{}',
  flags bit(8) default 0::bit(8)
); `

	createActivityPubActors = `
create table actors (
  "id" serial not null constraint actors_pkey primary key,
  "key" char(32) constraint actors_key_key unique,
  "account_id" int default NULL, -- the account for this actor
  "type" varchar, -- maybe enum
  "pub_id" varchar, -- the activitypub Object ID (APIURL/self/following/{key})
  "url" varchar, -- frontend reachable url
  "name" varchar,
  "preferred_username" varchar,
  "published" timestamp default CURRENT_TIMESTAMP,
  "updated" timestamp default CURRENT_TIMESTAMP,
  -- "inbox_id" int,
  "inbox" varchar,
  -- "outbox_id" int,
  "outbox" varchar,
  -- "liked_id" int,
  "liked" varchar,
  -- "followed_id" int,
  "followed" varchar,
  -- "following_id" int,
  "following" varchar
); `

	createActivityPubActivities = `
create table activities (
  "id" serial not null constraint activities_pkey primary key,
  "key" char(32) constraint activities_key_key unique,
  "pub_id" varchar, -- the activitypub Object ID
  "actor_id" int default NULL, -- the actor id, if this is a local activity
  "account_id" int default NULL, -- the account id, if this is a local actor
  "actor" varchar, -- the IRI of local or remote actor
  "object_id" int default NULL, -- the object id if it's a local object
  "item_id" int default NULL, -- the item id if it's a local object
  "object" varchar, -- the IRI of the local or remote object
  "published" timestamp default CURRENT_TIMESTAMP,
  "audience" jsonb -- the [to, cc, bto, bcc fields]
); `

	createActivityPubObjects = `
create table objects (
  "id" serial not null constraint objects_pkey primary key,
  "key" char(32) constraint objects_key_key unique,
  "pub_id" varchar, -- the activitypub Object ID
  "type" varchar, -- maybe enum
  "url" varchar,
  "name" varchar,
  "published" timestamp default CURRENT_TIMESTAMP,
  "updated" timestamp default CURRENT_TIMESTAMP
); `

	createActivityPubCollections = `
create table collections (
  "id" serial not null constraint collections_pkey primary key,
  "collection" varchar not null,
  "iri" varchar not null
);
`
)
