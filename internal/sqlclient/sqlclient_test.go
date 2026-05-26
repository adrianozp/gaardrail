package sqlclient

import "testing"

func TestWithDSNParam(t *testing.T) {
	cases := []struct {
		name string
		dsn  string
		want string
	}{
		{
			name: "no existing query string",
			dsn:  "root:root@tcp(localhost:3306)/gaardrail",
			want: "root:root@tcp(localhost:3306)/gaardrail?tls=true",
		},
		{
			name: "existing query string",
			dsn:  "root:root@tcp(localhost:3306)/gaardrail?parseTime=true",
			want: "root:root@tcp(localhost:3306)/gaardrail?parseTime=true&tls=true",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := withDSNParam(c.dsn, "tls", "true"); got != c.want {
				t.Errorf("withDSNParam() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRegisterMySQLTLS_BuiltinNames(t *testing.T) {
	cases := []struct {
		name string
		cfg  TLSConfig
		want string
	}{
		{name: "skip verify", cfg: TLSConfig{Enabled: true, SkipVerify: true}, want: "skip-verify"},
		{name: "default verification", cfg: TLSConfig{Enabled: true}, want: "true"},
		// SkipVerify takes precedence over other options.
		{name: "skip verify wins", cfg: TLSConfig{Enabled: true, SkipVerify: true, ServerName: "db"}, want: "skip-verify"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := registerMySQLTLS(c.cfg)
			if err != nil {
				t.Fatalf("registerMySQLTLS() unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("registerMySQLTLS() = %q, want %q", got, c.want)
			}
		})
	}
}

func TestRegisterMySQLTLS_CustomServerName(t *testing.T) {
	got, err := registerMySQLTLS(TLSConfig{Enabled: true, ServerName: "mysql.internal"})
	if err != nil {
		t.Fatalf("registerMySQLTLS() unexpected error: %v", err)
	}
	if got != "gaardrail-custom" {
		t.Errorf("registerMySQLTLS() = %q, want %q", got, "gaardrail-custom")
	}
}

func TestRegisterMySQLTLS_MissingCAFile(t *testing.T) {
	if _, err := registerMySQLTLS(TLSConfig{Enabled: true, CAFile: "/nonexistent/ca.pem"}); err == nil {
		t.Fatal("registerMySQLTLS() expected an error for a missing CA file, got nil")
	}
}
