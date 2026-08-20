package mysql

import (
	"fmt"
	"net/url"
	"strconv"
	"testing"
)

func TestDSNURLEscaping(t *testing.T) {
	endpoint := "endpoint"
	port := "3306"
	user := "username"
	rawPass := "password^"
	tls := "true"
	binlog := false
	dsn := DSN(user, rawPass, endpoint, port, tls, &binlog, nil)
	if dsn != fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=%s&sql_log_bin=%s",
		user,
		rawPass,
		endpoint,
		port,
		tls,
		strconv.FormatBool(binlog)) {
		t.Errorf("DSN string did not match expected output with URL encoded and binlog")
	}
}

func TestDSNURLEscapingWithoutBinLog(t *testing.T) {
	endpoint := "endpoint"
	port := "3306"
	user := "username"
	rawPass := "password^"
	tls := "true"
	dsn := DSN(user, rawPass, endpoint, port, tls, nil, nil)
	if dsn != fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=%s",
		user,
		rawPass,
		endpoint,
		port,
		tls) {
		t.Errorf("DSN string did not match expected output with URL encoded")
	}
}

func TestDSNSessionVariables(t *testing.T) {
	endpoint := "endpoint"
	port := "3306"
	user := "username"
	rawPass := "password"
	tls := "true"

	dsn := DSN(user, rawPass, endpoint, port, tls, nil, map[string]string{
		"wsrep_OSU_method": "'NBO'",
		"sql_mode":         "'STRICT_ALL_TABLES'",
	})

	want := fmt.Sprintf("%s:%s@tcp(%s:%s)/?tls=%s&sql_mode=%s&wsrep_OSU_method=%s",
		user, rawPass, endpoint, port, tls,
		url.QueryEscape("'STRICT_ALL_TABLES'"),
		url.QueryEscape("'NBO'"))

	if dsn != want {
		t.Errorf("DSN string with session variables did not match expected output.\ngot:  %s\nwant: %s", dsn, want)
	}
}

func TestDSNSessionVariablesDeterministicOrder(t *testing.T) {
	endpoint := "endpoint"
	port := "3306"
	user := "username"
	rawPass := "password"
	tls := "true"
	vars := map[string]string{
		"c": "3",
		"a": "1",
		"b": "2",
	}

	first := DSN(user, rawPass, endpoint, port, tls, nil, vars)
	for range 10 {
		if got := DSN(user, rawPass, endpoint, port, tls, nil, vars); got != first {
			t.Fatalf("DSN is not deterministic across calls with the same session variables.\nfirst: %s\ngot:   %s", first, got)
		}
	}
}
