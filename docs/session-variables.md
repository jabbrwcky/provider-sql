# MySQL session variables

provider-sql can set MySQL session variables on every physical connection it
opens against a server, via `sessionVariables` on a MySQL `ProviderConfig`.
This is useful for cluster-aware options that aren't exposed as regular DSN
parameters — for example, forcing Percona XtraDB Cluster / Galera nodes to use
a specific online schema upgrade method:

```sql
SET SESSION wsrep_OSU_method = 'NBO';
```

`sessionVariables` is available on `ProviderConfig` in both the cluster-scoped
(`mysql.sql.crossplane.io`) and namespaced (`mysql.sql.m.crossplane.io`) API
groups, as well as on `ClusterProviderConfig` for the namespaced API. It is
MySQL-specific; PostgreSQL and MSSQL ProviderConfigs do not have this field.

## Configuration

```yaml
apiVersion: mysql.sql.crossplane.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: MySQLConnectionSecret
    connectionSecretRef:
      namespace: default
      name: db-conn
  sessionVariables:
    wsrep_OSU_method: "'NBO'"
```

`sessionVariables` is a map of variable name to the literal right-hand side of
the `SET` statement. **String values must be quoted by you** (e.g.
`"'NBO'"`), since the value is used verbatim after `=` — the provider does not
add quotes for you.

Full examples:

- [`examples/cluster/mysql/config.yaml`](../examples/cluster/mysql/config.yaml)
- [`examples/namespaced/mysql/config.yaml`](../examples/namespaced/mysql/config.yaml)

## Behavior notes

- **Mechanism.** Each key/value pair is appended as an extra DSN query
  parameter. go-sql-driver/mysql runs any DSN parameter it doesn't recognize
  as `SET key = value` immediately after establishing a new physical
  connection, so this composes correctly with the
  [connection pool](connection-pool.md): every physical connection the pool
  opens gets the same session variables applied, not just the first.
- **Reserved names.** Do not use driver-recognized DSN parameter names (e.g.
  `tls`, `sql_log_bin`, or other go-sql-driver/mysql DSN params) as session
  variable keys — behavior is undefined if you do.
- **Pool cache key.** Session variables are baked into the DSN, and the
  connection pool is keyed by the exact DSN string. Changing
  `sessionVariables` therefore produces a new pool, isolated from the old one
  (as does any other DSN-affecting change, e.g. credential rotation).
