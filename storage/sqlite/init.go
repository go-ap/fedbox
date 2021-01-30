// +build storage_sqlite storage_all !sqlite_fs,!storage_boltdb,!storage_badger,!storage_pgx

package sqlite

const (

createActors = `
create table actors (
  "id" integer constraint actors_pkey primary key,
  "iri" varchar constraint actors_key_key unique,
  "type" varchar not null,
  "url" varchar,
  "name" varchar,
  "preferred_username" varchar,
  "published" timestamp default CURRENT_TIMESTAMP,
  "updated" timestamp default CURRENT_TIMESTAMP,
  "audience" blob, -- the [to, cc, bto, bcc fields]
  "raw" blob
);`

createActivities = `
create table activities (
  "id" integer constraint activities_pkey primary key,
  "iri" varchar constraint activities_key_key unique,
  "type" varchar not null,
  "url" varchar,
  "actor_id" int default NULL, -- the actor id, if this is a local activity
  "actor" varchar, -- the IRI of local or remote actor
  "object_id" int default NULL, -- the object id if it's a local object
  "object" varchar, -- the IRI of the local or remote object
  "published" timestamp default CURRENT_TIMESTAMP,
  "audience" blob, -- the [to, cc, bto, bcc fields]
  "raw" blob
);`

createObjects = `
create table objects (
  "id" integer constraint objects_pkey primary key,
  "iri" varchar constraint objects_key_key unique,
  "type" varchar not null,
  "url" varchar,
  "name" varchar,
  "published" timestamp default CURRENT_TIMESTAMP,
  "updated" timestamp default CURRENT_TIMESTAMP,
  "audience" blob, -- the [to, cc, bto, bcc fields]
  "raw" blob
);`

createCollections = `
create table collections (
	"id" integer constraint collections_pkey primary key, 
	"collection" varchar,
	"iri" varchar
);`
)
