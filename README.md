# open-pulumi-service

Open source backend service for [Pulumi](https://github.com/pulumi/pulumi)

This is a work in progress, but will (mostly) support the full state lifecycle of Pulumi stacks.

Currently requires postgres and Google KMS. No additional database support planned yet,
but other crypto providers will be added.

Environment variables:

| Var            | Default | Description                | Required |
|----------------|---------|----------------------------|----------|
| GCP_KMS_KEY_ID |         | GCP KMS key ID             | yes      |
| DATABASE_URL   |         | Postgres connection string | yes      |
| LISTEN_ADDRESS | 0.0.0.0 | HTTP listen  address       |          |
| LISTEN_PORT    | 8080    | HTTP listen port           |          |
