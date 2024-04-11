// Package apptest contains supporting code for running app layer tests.
package apptest

import (
	"context"
	"testing"

	eauth "encore.dev/beta/auth"
	eerrs "encore.dev/beta/errs"
	"github.com/ardanlabs/encore/app/api/errs"
	"github.com/ardanlabs/encore/app/api/mid"
	"github.com/ardanlabs/encore/business/api/auth"
)

// AppTable represent fields needed for running an app test.
type AppTable struct {
	Name    string
	Token   string
	ExpResp any
	ExcFunc func(ctx context.Context) any
	CmpFunc func(got any, exp any) string
}

// AuthHandler defines a function that can be called to handle authentication.
type AuthHandler func(ctx context.Context, ap *mid.AuthParams) (eauth.UID, *auth.Claims, error)

// =============================================================================

// AppTest contains functions for executing an app test.
type AppTest struct {
	handler AuthHandler
}

func New(handler AuthHandler) *AppTest {
	return &AppTest{
		handler: handler,
	}
}

// Run performs the actual test logic based on the table data.
func (at *AppTest) Run(t *testing.T, table []AppTable, testName string) {
	log := func(diff string, got any, exp any) {
		t.Log("DIFF")
		t.Logf("%s", diff)
		t.Log("GOT")
		t.Logf("%#v", got)
		t.Log("EXP")
		t.Logf("%#v", exp)
		t.Fatalf("Should get the expected response")
	}

	for _, tt := range table {
		f := func(t *testing.T) {
			ctx := context.Background()

			t.Log("Calling authHandler")
			ctx, err := at.authHandler(ctx, tt.Token)
			if err != nil {
				diff := tt.CmpFunc(err, tt.ExpResp)
				if diff != "" {
					log(diff, err, tt.ExpResp)
				}
				return
			}

			t.Log("Calling excFunc")
			got := tt.ExcFunc(ctx)

			diff := tt.CmpFunc(got, tt.ExpResp)
			if diff != "" {
				log(diff, got, tt.ExpResp)
			}
		}

		t.Run(testName+"-"+tt.Name, f)
	}
}

func (at *AppTest) authHandler(ctx context.Context, token string) (context.Context, error) {
	uid, claims, err := at.handler(ctx, &mid.AuthParams{
		Authorization: "Bearer " + token,
	})

	if err != nil {
		return ctx, err
	}

	return eauth.WithContext(ctx, uid, claims), nil
}

// =============================================================================

// CmpAppErrors compares two encore error values. If they are not equal, the
// reason is returned.
func CmpAppErrors(got any, exp any) string {
	expResp := exp.(*eerrs.Error)

	gotResp, exists := got.(*eerrs.Error)
	if !exists {
		return "no error occurred"
	}

	if gotResp.Code != expResp.Code {
		return "code does not match"
	}

	if gotResp.Message != expResp.Message {
		return "message does not match"
	}

	gotDetails := gotResp.Details.(errs.ExtraDetails)
	expDetails := expResp.Details.(errs.ExtraDetails)

	if gotDetails.HTTPStatus != expDetails.HTTPStatus {
		return "http status does not match"
	}

	if gotDetails.HTTPStatusCode != expDetails.HTTPStatusCode {
		return "http status code does not match"
	}

	return ""
}
