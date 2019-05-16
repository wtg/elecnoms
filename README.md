# elecnoms

elecnoms is a microservice that handles nominations for the [Elections](https://github.com/wtg/elections) website.

## Why

Nominations are a sufficiently detached section of the Elections site. Much of the Elections API needs to be overhauled, so we're going to start by implementing this new service in Go.

## Documentation

You need to have 3 environment variables setup to run this app:
CMS_TOKEN - The Union CMS API token, provided by the Union Sysadmins
DATABASE_URL - a standard USER:PASS@tcp(DB_IP_ADDRESS:DB_PORT)/DB_NAME database connection string
SESSION_SECRET - a random string the syncs with the equivalent setting on elections

Directions on how to run the app can be further derived from the Dockerfile.

Coming soon.
