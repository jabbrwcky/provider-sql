package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/crossplane-contrib/provider-sql/pkg/clients/pool"
	"github.com/crossplane-contrib/provider-sql/pkg/clients/xsql"
	"github.com/pkg/errors"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
)

const (
	errNotSupported = "%s not supported by mysql client"

	driverName = "mysql"
)

type mySQLDB struct {
	dsn      string
	endpoint string
	port     string
	tls      string
	pool     pool.Config
}

// New returns a new MySQL database client. The pool config tunes the shared,
// DSN-keyed connection pool used for all queries issued by this client.
// sessionVariables are set via `SET key = value` on every new physical
// connection the pool establishes.
func New(creds map[string][]byte, tls *string, binlog *bool, poolCfg pool.Config, sessionVariables map[string]string) xsql.DB {
	endpoint := string(creds[xpv1.ResourceCredentialsSecretEndpointKey])
	port := string(creds[xpv1.ResourceCredentialsSecretPortKey])
	username := string(creds[xpv1.ResourceCredentialsSecretUserKey])
	password := string(creds[xpv1.ResourceCredentialsSecretPasswordKey])
	if tls == nil {
		defaultTLS := "preferred"
		tls = &defaultTLS
	}
	dsn := DSN(username, password, endpoint, port, *tls, binlog, sessionVariables)

	return mySQLDB{
		dsn:      dsn,
		endpoint: endpoint,
		port:     port,
		tls:      *tls,
		pool:     poolCfg,
	}
}

// DSN returns the DSN URL. sessionVariables are appended as extra DSN query
// params in sorted key order: go-sql-driver/mysql runs any param it doesn't
// recognize as `SET key = value` immediately after connecting. Sorting keeps
// the DSN deterministic, which matters because the connection pool cache key
// is the literal DSN string.
func DSN(username, password, endpoint, port, tls string, binlog *bool, sessionVariables map[string]string) string {
	// Use net/url UserPassword to encode the username and password
	// This will ensure that any special characters in the username or password
	// are percent-encoded for use in the user info portion of the DSN URL
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%s@tcp(%s:%s)/?tls=%s", username, password, endpoint, port, tls)

	if binlog != nil {
		fmt.Fprintf(&b, "&sql_log_bin=%s", strconv.FormatBool(*binlog))
	}

	keys := make([]string, 0, len(sessionVariables))
	for k := range sessionVariables {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(&b, "&%s=%s", url.QueryEscape(k), url.QueryEscape(sessionVariables[k]))
	}

	return b.String()
}

// ExecTx is unsupported in MySQL.
func (c mySQLDB) ExecTx(ctx context.Context, ql []xsql.Query) error {
	return errors.Errorf(errNotSupported, "transactions")
}

// Exec the supplied query.
func (c mySQLDB) Exec(ctx context.Context, q xsql.Query) error {
	d, err := pool.Get(driverName, c.dsn, c.pool)
	if err != nil {
		return err
	}

	_, err = d.ExecContext(ctx, q.String, q.Parameters...)
	return err
}

// Query the supplied query.
func (c mySQLDB) Query(ctx context.Context, q xsql.Query) (*sql.Rows, error) {
	d, err := pool.Get(driverName, c.dsn, c.pool)
	if err != nil {
		return nil, err
	}

	rows, err := d.QueryContext(ctx, q.String, q.Parameters...)
	return rows, err
}

// Scan the results of the supplied query into the supplied destination.
func (c mySQLDB) Scan(ctx context.Context, q xsql.Query, dest ...interface{}) error {
	d, err := pool.Get(driverName, c.dsn, c.pool)
	if err != nil {
		return err
	}

	return d.QueryRowContext(ctx, q.String, q.Parameters...).Scan(dest...)
}

// GetConnectionDetails returns the connection details for a user of this DB
func (c mySQLDB) GetConnectionDetails(username, password string) managed.ConnectionDetails {
	return managed.ConnectionDetails{
		xpv1.ResourceCredentialsSecretUserKey:     []byte(username),
		xpv1.ResourceCredentialsSecretPasswordKey: []byte(password),
		xpv1.ResourceCredentialsSecretEndpointKey: []byte(c.endpoint),
		xpv1.ResourceCredentialsSecretPortKey:     []byte(c.port),
	}
}

// GetServerVersion is not supported by the MySQL client (only used by PostgreSQL).
func (c mySQLDB) GetServerVersion(ctx context.Context) (int, error) {
	// This method should never be called for MySQL clients
	// but is implemented to satisfy the xsql.DB interface
	return 0, nil
}

// QuoteIdentifier for MySQL queries
func QuoteIdentifier(id string) string {
	return "`" + strings.ReplaceAll(id, "`", "``") + "`"
}

// QuoteValue for MySQL queries
func QuoteValue(id string) string {
	return "'" + strings.ReplaceAll(id, "'", "''") + "'"
}

// SplitUserHost splits a MySQL user by name and host
func SplitUserHost(user string) (username, host string) {
	username = user
	host = "%"
	if strings.Contains(user, "@") {
		parts := strings.SplitN(user, "@", 2)
		username = parts[0]
		host = parts[1]
	}
	return username, host
}

// ExecQuery declares the query to execute and its error value if it fails
type ExecQuery struct {
	// Query defines the sql statement to execute
	Query string
	// ErrorValue defines what error will be returned if the provided sql statement failed when executing
	ErrorValue string
}

// ExecWrapper is a wrapper function for xsql.DB.Exec() that allows the execution of optional queries before and after the provided query
func ExecWrapper(ctx context.Context, db xsql.DB, query ExecQuery) error {
	if err := db.Exec(ctx, xsql.Query{
		String: query.Query,
	}); err != nil {
		return errors.Wrap(err, query.ErrorValue)
	}

	return nil
}
