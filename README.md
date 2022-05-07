# Directus v8 to Slack Webhook translator

Tiny service which translates incoming webhooks from [Directus v8](https://v8.docs.directus.io/)
to [Slack webhooks](https://api.slack.com/messaging/webhooks).

Expected format of incoming webhooks:

*URL*: `https://example.org/{userKey}/{objectTypeName}/{SLACK_WEBHOOK_SECRET}` where

- *https://example.org*: where this service is deployed
- *userKey*: secret for authentication, is checked against (hashed) KEY_HASH
- *objectTypeName*: e.g. "anlage", directus collection name (otherwise we have no way to know what object type was modified)
- *SLACK_WEBHOOK_SECRET*: secret URL path from Slack webhook URL, for example `T00000000/B00000000/XXXXXXXXXXXXXXXXXXXXXXXX` in the [Slack docs](https://api.slack.com/messaging/webhooks)

For example, the full URL for a `anlage` Webhook would be:
`https://example.org/my_secret_key/anlage/0O2JR73BDQ0M/9P4HEOWEVOX/F9NMD2SB3HA2ZXNBGIHAASHD37`

## Deployment

Deploy this as a Docker image.

```bash
docker build -t 'mova21-directus2slack-webhooks' .
docker run -it --rm -e KEY_HASH=$(go run ./cmd/hash/ MY_SECRET_KEY) mova21-directus2slack-webhooks
```

Config options are accepted as env vars. The only required one is `KEY_HASH`.
To hash a key, you can use `go run ./cmd/hash/ <KEY>`.

## Development

For testing, you can run this locally and use localtunnel or ngrok for forwarding from Directus.

```bash
KEY_HASH=$(go run ./cmd/hash/ MY_SECRET_KEY) go run ./cmd/server/
yarn add localtunnel
yarn run lt --port 8080
```
