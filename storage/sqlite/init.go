// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

const (

createAccounts = `
create table accounts (
  key text unique,
  handle text,
  created_at timestamp default current_timestamp,
  updated_at timestamp default current_timestamp,
  metadata jsonb default '{}',
  flags bit(8) default 0::bit(8)
);`

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
);`

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
);`

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
);`

createActivitypubActors = `
create table actors (
  "id" serial not null constraint actors_pkey primary key,
  "iri" char constraint actors_key_key unique,
  "type" varchar, -- maybe enum
  "url" varchar, -- frontend reachable url
  "name" char,
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
);`

createActivitypubActivities = `
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
);`

createActivitypubObjects = `
create table objects (
  "id" serial not null constraint objects_pkey primary key,
  "key" char(32) constraint objects_key_key unique,
  "pub_id" varchar, -- the activitypub Object ID
  "type" varchar, -- maybe enum
  "url" varchar,
  "name" varchar,
  "published" timestamp default CURRENT_TIMESTAMP,
  "updated" timestamp default CURRENT_TIMESTAMP
);`

createActivitypubCollections = `
create table collections (
)
`
)
