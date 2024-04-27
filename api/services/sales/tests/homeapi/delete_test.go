package home_test

import (
	"context"

	eerrs "encore.dev/beta/errs"
	"github.com/ardanlabs/encore/api/services/sales"
	"github.com/ardanlabs/encore/api/services/sales/tests/apitest"
	"github.com/ardanlabs/encore/app/api/errs"
	"github.com/google/go-cmp/cmp"
)

func homeDeleteOk(sd apitest.SeedData) []apitest.Table {
	table := []apitest.Table{
		{
			Name:    "user",
			Token:   sd.Users[0].Token,
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := sales.HomeDelete(ctx, sd.Users[0].Homes[1].ID.String()); err != nil {
					return err
				}

				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
		{
			Name:    "admin",
			Token:   sd.Admins[0].Token,
			ExpResp: nil,
			ExcFunc: func(ctx context.Context) any {
				if err := sales.HomeDelete(ctx, sd.Admins[0].Homes[1].ID.String()); err != nil {
					return err
				}

				return nil
			},
			CmpFunc: func(got any, exp any) string {
				return cmp.Diff(got, exp)
			},
		},
	}

	return table
}

func homeDeleteAuth(sd apitest.SeedData) []apitest.Table {
	table := []apitest.Table{
		{
			Name:    "emptytoken",
			Token:   "&nbsp;",
			ExpResp: errs.Newf(eerrs.Unauthenticated, "error parsing token: token contains an invalid number of segments"),
			ExcFunc: func(ctx context.Context) any {
				err := sales.HomeDelete(ctx, "")
				if err != nil {
					return err
				}

				return nil
			},
			CmpFunc: apitest.CmpAppErrors,
		},
		{
			Name:    "sig",
			Token:   sd.Users[0].Token + "A",
			ExpResp: errs.Newf(eerrs.Unauthenticated, "authentication failed : bindings results[[{[true] map[x:false]}]] ok[true]"),
			ExcFunc: func(ctx context.Context) any {
				err := sales.HomeDelete(ctx, "")
				if err != nil {
					return err
				}

				return nil
			},
			CmpFunc: apitest.CmpAppErrors,
		},
		{
			Name:    "wronguser",
			Token:   sd.Users[0].Token,
			ExpResp: errs.Newf(eerrs.Unauthenticated, "authorize: you are not authorized for that action, claims[[{USER}]] rule[rule_admin_or_subject]: rego evaluation failed : bindings results[[{[true] map[x:false]}]] ok[true]"),
			ExcFunc: func(ctx context.Context) any {
				err := sales.HomeDelete(ctx, sd.Admins[0].Homes[0].ID.String())
				if err != nil {
					return err
				}

				return nil
			},
			CmpFunc: apitest.CmpAppErrors,
		},
	}

	return table
}