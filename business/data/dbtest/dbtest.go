// Package dbtest contains supporting code for running tests that hit the DB.
package dbtest

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"net/mail"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/ardanlabs/encore/business/core/crud/delegate"
	"github.com/ardanlabs/encore/business/core/crud/user"
	"github.com/ardanlabs/encore/business/core/crud/user/stores/userdb"
	"github.com/ardanlabs/encore/business/data/appdb"
	"github.com/ardanlabs/encore/business/data/appdb/migrate"
	"github.com/ardanlabs/encore/business/data/sqldb"
	"github.com/ardanlabs/encore/business/web/auth"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jmoiron/sqlx"
)

// StartDB retrieves the database information.
func StartDB() (string, error) {
	var out bytes.Buffer
	cmd := exec.Command("encore", "db", "conn-uri", "--test", appdb.DBName)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("could not access the database information: %w", err)
	}

	url := out.String()
	url = strings.Trim(url, "\n")

	return url, nil
}

// StopDB stops a running database instance.
func StopDB() error {
	return nil
}

// Test owns state for running and shutting down tests.
type Test struct {
	DB       *sqlx.DB
	CoreAPIs CoreAPIs
	Teardown func()
	t        *testing.T
	V1       struct {
		Auth *auth.Auth
	}
}

// NewTest creates a test database inside a Docker container. It creates the
// required table structure but the database is otherwise empty. It returns
// the database to use as well as a function to call at the end of the test.
func NewTest(t *testing.T, url string, mainDBName string, testName string) *Test {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	dbM, err := sqldb.OpenTest(url)
	if err != nil {
		t.Fatalf("Opening database connection: %v", err)
	}

	if err := sqldb.StatusCheck(ctx, dbM); err != nil {
		t.Fatalf("status check database: %v", err)
	}

	const letterBytes = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 4)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	dbName := string(b)

	t.Logf("Creating database: %s", dbName)

	if _, err := dbM.ExecContext(context.Background(), "CREATE DATABASE "+dbName); err != nil {
		t.Fatalf("creating database %s: %v", dbName, err)
	}

	// -------------------------------------------------------------------------

	url = strings.Replace(url, mainDBName, dbName, 1)

	db, err := sqldb.OpenTest(url)
	if err != nil {
		t.Fatalf("Opening database connection: %v", err)
	}

	if err := migrate.Migrate(ctx, db); err != nil {
		t.Fatalf("Migrating error: %s", err)
	}

	if err := migrate.Seed(ctx, db); err != nil {
		t.Fatalf("Seeding error: %s", err)
	}

	// -------------------------------------------------------------------------

	coreAPIs := newCoreAPIs(db)

	// -------------------------------------------------------------------------

	cfg := auth.Config{
		DB:        db,
		KeyLookup: &keyStore{},
	}
	a, err := auth.New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// -------------------------------------------------------------------------

	// teardown is the function that should be invoked when the caller is done
	// with the database.
	teardown := func() {
		t.Helper()

		t.Logf("Dropping database: %s", dbName)

		db.Close()
		defer dbM.Close()

		if _, err := dbM.ExecContext(context.Background(), "DROP DATABASE "+dbName); err != nil {
			fmt.Printf("dropping database %s: %v", dbName, err)
		}
	}

	test := Test{
		DB:       db,
		CoreAPIs: coreAPIs,
		Teardown: teardown,
		t:        t,
		V1: struct {
			Auth *auth.Auth
		}{
			Auth: a,
		},
	}

	return &test
}

// TokenV1 generates an authenticated token for a user.
func (test *Test) TokenV1(email string, pass string) string {
	addr, _ := mail.ParseAddress(email)

	store := userdb.NewStore(test.DB)
	dbUsr, err := store.QueryByEmail(context.Background(), *addr)
	if err != nil {
		return ""
	}

	claims := auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   dbUsr.ID.String(),
			Issuer:    "service project",
			ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		},
		Roles: dbUsr.Roles,
	}

	token, err := test.V1.Auth.GenerateToken(kid, claims)
	if err != nil {
		test.t.Fatal(err)
	}

	return token
}

// StringPointer is a helper to get a *string from a string. It is in the tests
// package because we normally don't want to deal with pointers to basic types
// but it's useful in some tests.
func StringPointer(s string) *string {
	return &s
}

// IntPointer is a helper to get a *int from a int. It is in the tests package
// because we normally don't want to deal with pointers to basic types but it's
// useful in some tests.
func IntPointer(i int) *int {
	return &i
}

// FloatPointer is a helper to get a *float64 from a float64. It is in the tests
// package because we normally don't want to deal with pointers to basic types
// but it's useful in some tests.
func FloatPointer(f float64) *float64 {
	return &f
}

// CoreAPIs represents all the core api's needed for testing.
type CoreAPIs struct {
	Delegate *delegate.Delegate
	User     *user.Core
}

func newCoreAPIs(db *sqlx.DB) CoreAPIs {
	delegate := delegate.New()
	usrCore := user.NewCore(delegate, userdb.NewStore(db))

	return CoreAPIs{
		Delegate: delegate,
		User:     usrCore,
	}
}

type keyStore struct{}

func (ks *keyStore) PrivateKey(kid string) (string, error) {
	return privateKeyPEM, nil
}

func (ks *keyStore) PublicKey(kid string) (string, error) {
	return publicKeyPEM, nil
}

const (
	kid = "s4sKIjD9kIRjxs2tulPqGLdxSfgPErRN1Mu3Hd9k9NQ"

	privateKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEpQIBAAKCAQEAvMAHb0IoLvoYuW2kA+LTmnk+hfnBq1eYIh4CT/rMPCxgtzjq
U0guQOMnLg69ydyA5uu37v6rbS1+stuBTEiMQl/bxAhgLkGrUhgpZ10Bt6GzSEgw
QNloZoGaxe4p20wMPpT4kcMKNHkQds3uONNcLxPUmfjbbH64g+seg28pbgQPwKFK
tF7bIsOBgz0g5Ptn5mrkdzqMPUSy9k9VCu+R42LH9c75JsRzz4FeN+VzwMAL6yQn
ZvOi7/zOgNyxeVia8XVKykrnhgcpiOn5oaLRBzQGN00Z7TuBRIfDJWU21qQN4Cq7
keZmMP4gqCVWjYneK4bzrG/+H2w9BJ2TsmMGvwIDAQABAoIBAFQmQKpHkmavNYql
6POaksBRwaA1YzSijr7XJizGIXvKRSwqgb2zdnuTSgpspAx09Dr/aDdy7rZ0DAJt
fk2mInINDottOIQm3txwzTS58GQQAT/+fxTKWJMqwPfxYFPWqbbU76T8kXYna0Gs
OcK36GdMrgIfQqQyMs0Na8MpMg1LmkAxuqnFCXS/NMyKl9jInaaTS+Kz+BSzUMGQ
zebfLFsf2N7sLZuimt9zlRG30JJTfBlB04xsYMo734usA2ITe8U0XqG6Og0qc6ev
6lsoM8hpvEUsQLcjQQ5up7xx3S2stZJ8o0X8GEX5qUMaomil8mZ7X5xOlEqf7p+v
lXQ46cECgYEA2lbZQON6l3ZV9PCn9j1rEGaXio3SrAdTyWK3D1HF+/lEjClhMkfC
XrECOZYj+fiI9n+YpSog+tTDF7FTLf7VP21d2gnhQN6KAXUnLIypzXxodcC6h+8M
ZGJh/EydLvC7nPNoaXx96bohxzS8hrOlOlkCbr+8gPYKf8qkbe7HyxECgYEA3U6e
x9g4FfTvI5MGrhp2BIzoRSn7HlNQzjJ71iMHmM2kBm7TsER8Co1PmPDrP8K/UyGU
Q25usTsPSrHtKQEV6EsWKaP/6p2Q82sDkT9bZlV+OjRvOfpdO5rP6Q95vUmMGWJ/
S6oimbXXL8p3gDafw3vC1PCAhoaxMnGyKuZwlM8CgYEAixT1sXr2dZMg8DV4mMfI
8pqXf+AVyhWkzsz+FVkeyAKiIrKdQp0peI5C/5HfevVRscvX3aY3efCcEfSYKt2A
07WEKkdO4LahrIoHGT7FT6snE5NgfwTMnQl6p2/aVLNun20CHuf5gTBbIf069odr
Af7/KLMkjfWs/HiGQ6zuQjECgYEAv+DIvlDz3+Wr6dYyNoXuyWc6g60wc0ydhQo0
YKeikJPLoWA53lyih6uZ1escrP23UOaOXCDFjJi+W28FR0YProZbwuLUoqDW6pZg
U3DxWDrL5L9NqKEwcNt7ZIDsdnfsJp5F7F6o/UiyOFd9YQb7YkxN0r5rUTg7Lpdx
eMyv0/UCgYEAhX9MPzmTO4+N8naGFof1o8YP97pZj0HkEvM0hTaeAQFKJiwX5ijQ
xumKGh//G0AYsjqP02ItzOm2mWnbI3FrNlKmGFvR6VxIZMOyXvpLofHucjJ5SWli
eYjPklKcXaMftt1FVO4n+EKj1k1+Tv14nytq/J5WN+r4FBlNEYj/6vg=
-----END PRIVATE KEY-----
`
	publicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvMAHb0IoLvoYuW2kA+LT
mnk+hfnBq1eYIh4CT/rMPCxgtzjqU0guQOMnLg69ydyA5uu37v6rbS1+stuBTEiM
Ql/bxAhgLkGrUhgpZ10Bt6GzSEgwQNloZoGaxe4p20wMPpT4kcMKNHkQds3uONNc
LxPUmfjbbH64g+seg28pbgQPwKFKtF7bIsOBgz0g5Ptn5mrkdzqMPUSy9k9VCu+R
42LH9c75JsRzz4FeN+VzwMAL6yQnZvOi7/zOgNyxeVia8XVKykrnhgcpiOn5oaLR
BzQGN00Z7TuBRIfDJWU21qQN4Cq7keZmMP4gqCVWjYneK4bzrG/+H2w9BJ2TsmMG
vwIDAQAB
-----END PUBLIC KEY-----`
)
