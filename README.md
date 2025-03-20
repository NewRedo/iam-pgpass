# iam-pgpass

A Unix Pipe Server to dynamically create PGPASS files

## Usage

```
iam-pgpass <file path>
```

The server must be run with an argument which is either a non-existent file or an existing fifo 
pipe. It will create the pipe if necessary upon start up.

When connecting to your PostgreSQL RDS database, when an IAM user has been set up, export the
environment variable `PGPASSFILE` to the named pipe and this will allow your user to connect to
the database.

## Configuration

The server expects the usual AWS SDK environment variables to be set to allow access to the AWS
APIs, for example `AWS_PROFILE`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` or 
`AWS_WEB_IDENTITY_TOKEN_FILE`.

The AWS Region must be configured, either through the profile configuration, or through the
`AWS_REGION` or `AWS_DEFAULT_REGION` environment variables.

The PostgreSQL server details must also be set through the standard `PG` environment variables;
namely:

* `PGHOST`; the hostname to connect to (defaults to `localhost`)
* `PGPORT`; the port to connect to (defaults to `5432`)
* `PGUSER`; the username to use (defaults to `postgres`)
* `PGDATABASE`; the database to connect to (defaults to `postgres`)

These variables, along with the `PGPASSFILE` variable will also allow you to connect to your RDS
database.

## Development

The project is a golang module; use your standard Go tooling to build the binary. 

There is also a Nix flake-based development environment which sets up Go for you. Either run

```
nix-develop
```

in the project, or use `direnv` to automatically create the environment for you.

A flake-based build is also available for the following architectures:

* aarch64-darwin
* aarch64-linux
* x86_64-darwin
* x86_64-linux

Run `nix build` to build the package with nix on these systems. The output is placed in 
`result/bin/iam-pgpass`.

### Docker

A nix-based docker build is also under development. Build the image with `nix build .#docker` and
load into your Docker engine with `docker load -i result`.
