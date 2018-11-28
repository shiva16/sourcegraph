package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/pkg/actor"
	"github.com/sourcegraph/sourcegraph/pkg/errcode"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc"
)

func TestUpdateUser(t *testing.T) {
	const (
		wantUsername     = "u"
		wantUserID       = 1
		wantAuthedUserID = 2
	)

	mockUsersGetByID := func() {
		// This should always pass, because in the impl we just were able to retrieve the user.
		db.Mocks.Users.GetByID = func(_ context.Context, userID int32) (*types.User, error) {
			return &types.User{ID: userID, Username: wantUsername}, nil
		}
	}
	mockUsersUpdate := func() {
		db.Mocks.Users.Update = func(int32, db.UserUpdate) error { return nil }
	}
	mockNewUser := func(t *testing.T) {
		db.Mocks.ExternalAccounts.LookupUserAndSave = func(a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (userID int32, err error) {
			return 0, &errcode.Mock{IsNotFound: true}
		}
	}
	mockExistingUser := func(t *testing.T) {
		db.Mocks.ExternalAccounts.LookupUserAndSave = func(a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (userID int32, err error) {
			return wantUserID, nil
		}
	}

	t.Run("new user, new external account, createIfNotExist=true", func(t *testing.T) {
		var calledCreateUserAndSave bool
		mockUsersUpdate()
		mockNewUser(t)
		db.Mocks.ExternalAccounts.CreateUserAndSave = func(u db.NewUser, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (createdUserID int32, err error) {
			calledCreateUserAndSave = true
			return wantUserID, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()
		userID, _, err := UpdateUser(context.Background(), db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, true)
		if err != nil {
			t.Fatal(err)
		}
		if userID != wantUserID {
			t.Errorf("got %d, want %d", userID, wantUserID)
		}
		if !calledCreateUserAndSave {
			t.Error("!calledCreateUserAndSave")
		}
	})

	t.Run("new user, new external account, createIfNotExist=false", func(t *testing.T) {
		var calledCreateUserAndSave bool
		mockUsersUpdate()
		mockNewUser(t)
		db.Mocks.ExternalAccounts.CreateUserAndSave = func(u db.NewUser, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (createdUserID int32, err error) {
			calledCreateUserAndSave = true
			return wantUserID, nil
		}
		defer func() { db.Mocks = db.MockStores{} }()
		_, _, err := UpdateUser(context.Background(), db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, false)
		if !errcode.IsNotFound(err) {
			t.Errorf("Expected \"not found\" error, but got %#v", err)
		}
		if calledCreateUserAndSave {
			t.Error("calledCreateUserAndSave")
		}
	})

	t.Run("new user, existing external account, createIfNotExist=true", func(t *testing.T) {
		var calledCreateUserAndSave bool
		mockUsersUpdate()
		mockNewUser(t)
		wantErr := errors.New("x")
		db.Mocks.ExternalAccounts.CreateUserAndSave = func(u db.NewUser, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (createdUserID int32, err error) {
			calledCreateUserAndSave = true
			return 0, wantErr
		}
		defer func() { db.Mocks = db.MockStores{} }()
		if _, _, err := UpdateUser(context.Background(), db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, true); err != wantErr {
			t.Fatalf("got err %q, want %q", err, wantErr)
		}
		if !calledCreateUserAndSave {
			t.Error("!calledCreateUserAndSave")
		}
	})

	t.Run("new user, existing external account, createIfNotExist=false", func(t *testing.T) {
		var calledCreateUserAndSave bool
		mockUsersUpdate()
		mockNewUser(t)
		db.Mocks.ExternalAccounts.CreateUserAndSave = func(u db.NewUser, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) (createdUserID int32, err error) {
			calledCreateUserAndSave = true
			return 0, errors.New("x")
		}
		defer func() { db.Mocks = db.MockStores{} }()
		_, _, err := UpdateUser(context.Background(), db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, false)
		if !errcode.IsNotFound(err) {
			t.Errorf("Expected \"not found\" error, but got %#v", err)
		}
		if calledCreateUserAndSave {
			t.Error("calledCreateUserAndSave")
		}
	})

	for _, createIfNotExist := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing user, new external account, createIfNotExist=%v", createIfNotExist), func(t *testing.T) {
			var calledAssociateUserAndSave bool
			mockUsersGetByID()
			mockUsersUpdate()
			mockExistingUser(t)
			db.Mocks.ExternalAccounts.AssociateUserAndSave = func(userID int32, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) error {
				if userID != wantAuthedUserID {
					t.Errorf("got %d, want %d", userID, wantAuthedUserID)
				}
				calledAssociateUserAndSave = true
				return nil
			}
			defer func() { db.Mocks = db.MockStores{} }()
			ctx := actor.WithActor(context.Background(), &actor.Actor{UID: wantAuthedUserID})
			userID, _, err := UpdateUser(ctx, db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, createIfNotExist)
			if err != nil {
				t.Fatal(err)
			}
			if userID != wantAuthedUserID {
				t.Errorf("got %d, want %d", userID, wantUserID)
			}
			if !calledAssociateUserAndSave {
				t.Error("!calledAssociateUserAndSave")
			}
		})
	}

	for _, createIfNotExist := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing user, existing (conflicting) external account, createIfNotExist=%v", createIfNotExist), func(t *testing.T) {
			var calledAssociateUserAndSave bool
			mockUsersGetByID()
			mockUsersUpdate()
			mockExistingUser(t)
			wantErr := errors.New("x")
			db.Mocks.ExternalAccounts.AssociateUserAndSave = func(userID int32, a extsvc.ExternalAccountSpec, d extsvc.ExternalAccountData) error {
				calledAssociateUserAndSave = true
				return wantErr
			}
			defer func() { db.Mocks = db.MockStores{} }()
			ctx := actor.WithActor(context.Background(), &actor.Actor{UID: wantAuthedUserID})
			if _, _, err := UpdateUser(ctx, db.NewUser{Username: wantUsername}, extsvc.ExternalAccountSpec{}, extsvc.ExternalAccountData{}, createIfNotExist); err != wantErr {
				t.Fatalf("got err %q, want %q", err, wantErr)
			}
			if !calledAssociateUserAndSave {
				t.Error("!calledAssociateUserAndSave")
			}
		})
	}
}
