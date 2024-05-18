// Package usercache contains user related CRUD functionality with caching.
package usercache

import (
	"context"
	"net/mail"
	"sync"
	"time"

	"encore.dev/rlog"
	"github.com/ardanlabs/encore/business/domain/userbus"
	"github.com/ardanlabs/encore/business/sdk/order"
	"github.com/ardanlabs/encore/business/sdk/page"
	"github.com/ardanlabs/encore/business/sdk/transaction"
	"github.com/creativecreature/sturdyc"
	"github.com/google/uuid"
)

// Store manages the set of APIs for user data and caching.
type Store struct {
	log    rlog.Ctx
	storer userbus.Storer
	cache  *sturdyc.Client[userbus.User]
	mu     sync.RWMutex
}

// NewStore constructs the api for data and caching access.
func NewStore(log rlog.Ctx, storer userbus.Storer, ttl time.Duration) *Store {
	const capacity = 10000
	const numShards = 10
	const evictionPercentage = 10

	return &Store{
		log:    log,
		storer: storer,
		cache:  sturdyc.New[userbus.User](capacity, numShards, ttl, evictionPercentage),
	}
}

// NewWithTx constructs a new Store value replacing the sqlx DB
// value with a sqlx DB value that is currently inside a transaction.
func (s *Store) NewWithTx(tx transaction.CommitRollbacker) (userbus.Storer, error) {
	return s.storer.NewWithTx(tx)
}

// Create inserts a new user into the database.
func (s *Store) Create(ctx context.Context, usr userbus.User) error {
	if err := s.storer.Create(ctx, usr); err != nil {
		return err
	}

	s.writeCache(usr)

	return nil
}

// Update replaces a user document in the database.
func (s *Store) Update(ctx context.Context, usr userbus.User) error {
	if err := s.storer.Update(ctx, usr); err != nil {
		return err
	}

	s.writeCache(usr)

	return nil
}

// Delete removes a user from the database.
func (s *Store) Delete(ctx context.Context, usr userbus.User) error {
	if err := s.storer.Delete(ctx, usr); err != nil {
		return err
	}

	s.deleteCache(usr)

	return nil
}

// Query retrieves a list of existing users from the database.
func (s *Store) Query(ctx context.Context, filter userbus.QueryFilter, orderBy order.By, page page.Page) ([]userbus.User, error) {
	return s.storer.Query(ctx, filter, orderBy, page)
}

// Count returns the total number of cards in the DB.
func (s *Store) Count(ctx context.Context, filter userbus.QueryFilter) (int, error) {
	return s.storer.Count(ctx, filter)
}

// QueryByID gets the specified user from the database.
func (s *Store) QueryByID(ctx context.Context, userID uuid.UUID) (userbus.User, error) {
	cachedUsr, ok := s.readCache(userID.String())
	if ok {
		return cachedUsr, nil
	}

	usr, err := s.storer.QueryByID(ctx, userID)
	if err != nil {
		return userbus.User{}, err
	}

	s.writeCache(usr)

	return usr, nil
}

// QueryByIDs gets the specified users from the database.
func (s *Store) QueryByIDs(ctx context.Context, userIDs []uuid.UUID) ([]userbus.User, error) {
	usr, err := s.storer.QueryByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	return usr, nil
}

// QueryByEmail gets the specified user from the database by email.
func (s *Store) QueryByEmail(ctx context.Context, email mail.Address) (userbus.User, error) {
	cachedUsr, ok := s.readCache(email.Address)
	if ok {
		return cachedUsr, nil
	}

	usr, err := s.storer.QueryByEmail(ctx, email)
	if err != nil {
		return userbus.User{}, err
	}

	s.writeCache(usr)

	return usr, nil
}

// readCache performs a safe search in the cache for the specified key.
func (s *Store) readCache(key string) (userbus.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	usr, exists := s.cache.Get(key)
	if !exists {
		return userbus.User{}, false
	}

	return usr, true
}

// writeCache performs a safe write to the cache for the specified userbus.
func (s *Store) writeCache(usr userbus.User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Set(usr.ID.String(), usr)
	s.cache.Set(usr.Email.Address, usr)
}

// deleteCache performs a safe removal from the cache for the specified userbus.
func (s *Store) deleteCache(usr userbus.User) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.cache.Delete(usr.ID.String())
	s.cache.Delete(usr.Email.Address)
}
