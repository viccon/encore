package homeapi

import (
	"context"
	"fmt"
	"time"

	eerrs "encore.dev/beta/errs"
	"github.com/ardanlabs/encore/business/api/errs"
	"github.com/ardanlabs/encore/business/api/mid"
	"github.com/ardanlabs/encore/business/core/crud/home"
	"github.com/ardanlabs/encore/foundation/validate"
)

// QueryParams represents the set of possible query strings.
type QueryParams struct {
	Page             int    `query:"page"`
	Rows             int    `query:"rows"`
	OrderBy          string `query:"orderBy"`
	ID               string `query:"home_id"`
	UserID           string `query:"user_id"`
	Type             string `query:"type"`
	StartCreatedDate string `query:"start_created_date"`
	EndCreatedDate   string `query:"end_created_date"`
}

// AppAddress represents information about an individual address.
type AppAddress struct {
	Address1 string `json:"address1"`
	Address2 string `json:"address2"`
	ZipCode  string `json:"zipCode"`
	City     string `json:"city"`
	State    string `json:"state"`
	Country  string `json:"country"`
}

// AppHome represents information about an individual home.
type AppHome struct {
	ID          string     `json:"id"`
	UserID      string     `json:"userID"`
	Type        string     `json:"type"`
	Address     AppAddress `json:"address"`
	DateCreated string     `json:"dateCreated"`
	DateUpdated string     `json:"dateUpdated"`
}

func toAppHome(hme home.Home) AppHome {
	return AppHome{
		ID:     hme.ID.String(),
		UserID: hme.UserID.String(),
		Type:   hme.Type.Name(),
		Address: AppAddress{
			Address1: hme.Address.Address1,
			Address2: hme.Address.Address2,
			ZipCode:  hme.Address.ZipCode,
			City:     hme.Address.City,
			State:    hme.Address.State,
			Country:  hme.Address.Country,
		},
		DateCreated: hme.DateCreated.Format(time.RFC3339),
		DateUpdated: hme.DateUpdated.Format(time.RFC3339),
	}
}

func toAppHomes(homes []home.Home) []AppHome {
	items := make([]AppHome, len(homes))
	for i, hme := range homes {
		items[i] = toAppHome(hme)
	}

	return items
}

// AppNewAddress defines the data needed to add a new address.
type AppNewAddress struct {
	Address1 string `json:"address1" validate:"required,min=1,max=70"`
	Address2 string `json:"address2" validate:"omitempty,max=70"`
	ZipCode  string `json:"zipCode" validate:"required,numeric"`
	City     string `json:"city" validate:"required"`
	State    string `json:"state" validate:"required,min=1,max=48"`
	Country  string `json:"country" validate:"required,iso3166_1_alpha2"`
}

// AppNewHome defines the data needed to add a new home.
type AppNewHome struct {
	Type    string        `json:"type" validate:"required"`
	Address AppNewAddress `json:"address"`
}

func toCoreNewHome(ctx context.Context, app AppNewHome) (home.NewHome, error) {
	userID, err := mid.GetUserID(ctx)
	if err != nil {
		return home.NewHome{}, fmt.Errorf("getuserid: %w", err)
	}

	typ, err := home.ParseType(app.Type)
	if err != nil {
		return home.NewHome{}, fmt.Errorf("parse: %w", err)
	}

	hme := home.NewHome{
		UserID: userID,
		Type:   typ,
		Address: home.Address{
			Address1: app.Address.Address1,
			Address2: app.Address.Address2,
			ZipCode:  app.Address.ZipCode,
			City:     app.Address.City,
			State:    app.Address.State,
			Country:  app.Address.Country,
		},
	}

	return hme, nil
}

// Validate checks if the data in the model is considered clean.
func (app AppNewHome) Validate() error {
	if err := validate.Check(app); err != nil {
		return errs.Newf(eerrs.FailedPrecondition, "validate: %s", err)
	}

	return nil
}

// AppUpdateAddress defines the data needed to update an address.
type AppUpdateAddress struct {
	Address1 *string `json:"address1" validate:"omitempty,min=1,max=70"`
	Address2 *string `json:"address2" validate:"omitempty,max=70"`
	ZipCode  *string `json:"zipCode" validate:"omitempty,numeric"`
	City     *string `json:"city"`
	State    *string `json:"state" validate:"omitempty,min=1,max=48"`
	Country  *string `json:"country" validate:"omitempty,iso3166_1_alpha2"`
}

// Validate checks the data in the model is considered clean.
func (app AppUpdateAddress) Validate() error {
	if err := validate.Check(app); err != nil {
		return err
	}

	return nil
}

// AppUpdateHome defines the data needed to update a home.
type AppUpdateHome struct {
	Type    *string           `json:"type"`
	Address *AppUpdateAddress `json:"address"`
}

func toCoreUpdateHome(app AppUpdateHome) (home.UpdateHome, error) {
	var typ home.Type
	if app.Type != nil {
		var err error
		typ, err = home.ParseType(*app.Type)
		if err != nil {
			return home.UpdateHome{}, fmt.Errorf("parse: %w", err)
		}
	}

	core := home.UpdateHome{
		Type: &typ,
	}

	if app.Address != nil {
		core.Address = &home.UpdateAddress{
			Address1: app.Address.Address1,
			Address2: app.Address.Address2,
			ZipCode:  app.Address.ZipCode,
			City:     app.Address.City,
			State:    app.Address.State,
			Country:  app.Address.Country,
		}
	}

	return core, nil
}

// Validate checks the data in the model is considered clean.
func (app AppUpdateHome) Validate() error {
	if err := validate.Check(app); err != nil {
		return errs.Newf(eerrs.FailedPrecondition, "validate: %s", err)
	}

	return nil
}
