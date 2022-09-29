# vault-plugin-kuma

HashiCorp Vault Secrets Engine to mage Kuma Service Mesh Tokens

For an example demo setup of Vault and Kuma that uses this plugin, please see the following repo:

[https://github.com/gregoryhunt/demo-kuma-vault](https://github.com/gregoryhunt/demo-kuma-vault)

## Installing the Plugin

To install the plugin to your Vault server, first download the latest release of this plugin to your Vault servers plugin folder and 
run the following command to register it with Vault.

```
vault write sys/plugins/catalog/secret/vault-plugin-kuma \
    sha_256="$(sha256sum /plugins/vault-plugin-kuma | cut -d " " -f 1)" \
    command="vault-plugin-kuma"
```

## Enabling the plugin

Before a custom plugin can be used it needs to be enabled and configured, you can enable a plugin multiple times allowing you
to manage multiple Kuma service meshes with a single Vault server.

```shell
vault secrets enable -path=kuma vault-plugin-kuma
```

Once enabled the plugin can be configured, there are two parameters that need to be set in order to configure the plugin.

* url - the URL of the Kuma Control Pane
* token - a Kuma admin token that has the permission to generate other Kuma tokens.

```shell
vault write kuma/config url=http://your.kuma.control.pane.address token=xxdfdfdfu88888
```

## Generating User Tokens

To generate Kuma Control Pane user tokens you need to define a role, besides the default Vault configuration a role for a user token
has the following parameters.

* token - the name that will be encoded into the token.
* mesh - the mesh that the token is scoped for.
* ttl - the lease for the token.
* max_ttl - the maximum duration a token can exists, this is encoded as the `exp` parameter inside the JWT.
* groups - the permissions that this user token has.

```shell
vault write kuma/roles/kuma-admin-role \
  token_name=jerry \
  mesh=default \
  ttl=1h \
  max_ttl=24h \
  groups="mesh-system:admin"
```

Once the role has been created the token can be generated with the following command.

```shell
vault read kuma/creds/kuma-admin-role
```

The response will look something like the following, the token parameter will contain a valid JWT that can be used to access the Kuma control pane.

```shell
Key                Value
---                -----
lease_id           kuma-guide/creds/kuma-admin-role/u8LuR60iV7ZJ0jS0xYGjQJAy
lease_duration     1h
lease_renewable    true
token              eyJhbGciOiJSUzI1NiIsImtpZCI6IjEiLCJ0eXAiOiJKV1QifQ.eyJOYW1lIjoiamVycnkiLCJHcm91cHMiOlsibWVzaC1zeXN0ZW06YWRtaW4iXSwiZXhwIjoxNjY0Mjg5ODQ1LCJuYmYiOjE2NjQyMDMxNDUsImlhdCI6MTY2NDIwMzQ0NSwianRpIjoiZjMyNmU3ZDUtMDI0NC00MWRhLTlhNjgtNGQwNWQyNmQ0MGYwIn0.oRKlvAQMNd8ytgHahcR7VBOkS9Y-Ir9qf0I41vy8mL68OZatanLdR3QnOomF-8TJ8USV3W8DPi9iRpjs7c3FJL_4qsBHI19ZH37C2RxZJvYUJMefZWszSnuwlccvNns6YRMTAu_4DRfIZgYwR3T2Wn6shMyVkQu92cxHCBoaoL-9aiRtvmVSCovglXPGwJ_PpXM53TbdBFtAvTtwnqrVSez4Amp6C4nKGqdy0AuXdQ-mHHmpeHfVFLlMPxwBfoNopf-NfucH6pbehaWyJhN4uDJjNnboJXltFl4l_oacIOeDclO93dG4nmQQUU4SsRainUVcCUZCDWFk8bWYS9DdfA
```

## Generating Dataplane Tokens

Like user tokens, to generate tokens that can be used to register Kuma Dataplanes or services, you need to define a role, a role for a dataplane token
has the following parameters.

* token - the name that will be encoded into the token.
* mesh - the mesh that the token is scoped for.
* ttl - the lease for the token.
* max_ttl - the maximum duration a token can exists, this is encoded as the `exp` parameter inside the JWT.
* tags - the capabilites that the token has, the following example defines a token that could register the payments service.

```shell
vault write kuma-guide/roles/payments-role \
  token_name=payments \
  mesh=default \
  ttl=1h \
  max_ttl=168h \
  tags="kuma.io/service=payments"
```

## Token Lifecycle

When Vault generates a Kuma token it creates it with an expiration date equal to the `max_ttl` plus the creation time. However, in addition
to this every generated token as a `lease`.

[https://www.vaultproject.io/docs/concepts/lease](https://www.vaultproject.io/docs/concepts/lease)

The lease allows Vault to manage the short term lifecycle of a token, by renewing the lease you can continue to use a token until it eventually
expires. If a lease is not renewed then Vault assumes that the token is no longer being used and will automatically add it to Kumas token revocation
list. This allows you to to find a balance between token security and the churn of regenerating tokens and restarting dataplanes when the tokens expire.

Tools like Vault Agent automatically renew the lease for secrets it obtains, this gives you a fully automated and secure process to manage the lifecycle
of your tokens.

In the instance that a Token needs to be manually revoked, an operator can manually do this by using the `vault lease revoke` command, this way
you do not need to keep track of the JTIs for any tokens that have been created and manually add them to the Kuma revocation secrets.
If a token has been added to Kuma's revocation secrets it will eventually expire and be unusable, Vault automatically keeps track of the expiries of
any tokens that have been added to the revocation list. When a token eventually expires, Vault will automatically remove it from the revocation 
secrets keeping this list clean and concise.
