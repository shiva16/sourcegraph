package auth

import (
	"context"
	"fmt"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/db"
	"github.com/sourcegraph/sourcegraph/pkg/actor"
	"github.com/sourcegraph/sourcegraph/pkg/errcode"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

// MockUpdateUser is used in tests to mock UpdateUser.
var MockUpdateUser func(db.NewUser, extsvc.ExternalAccountSpec) (int32, error)

// UpdateUser creates or updates a user using the provided information, looking up a user by
// the external account provided.
//
// 🚨 SECURITY: The safeErrMsg is an error message that can be shown to unauthenticated users to
// describe the problem. The err may contain sensitive information and should only be written to the
// server error logs, not to the HTTP response to shown to unauthenticated users.
func UpdateUser(ctx context.Context, newOrUpdatedUser db.NewUser, externalAccount extsvc.ExternalAccountSpec, data extsvc.ExternalAccountData, createIfNotExist bool) (
	userID int32, safeErrMsg string, err error,
) {
	if MockUpdateUser != nil {
		userID, err = MockUpdateUser(newOrUpdatedUser, externalAccount)
		return userID, "", err
	}

	if actor := actor.FromContext(ctx); actor.IsAuthenticated() {
		// There is already an authenticated actor, so this external account will be added to
		// the existing user account.
		userID = actor.UID

		if err := db.ExternalAccounts.AssociateUserAndSave(ctx, userID, externalAccount, data); err != nil {
			safeErrMsg = "Unexpected error associating the external account with your Sourcegraph user. The most likely cause for this problem is that another Sourcegraph user is already linked with this external account. A site admin or the other user can unlink the account to fix this problem."
			return 0, safeErrMsg, err
		}
	} else {
		userID, err = db.ExternalAccounts.LookupUserAndSave(ctx, externalAccount, data)
		if err != nil {
			if !errcode.IsNotFound(err) {
				return 0, "Unexpected error looking up the Sourcegraph user account associated with the external account. Ask a site admin for help.", err
			}
			// err is "not found"
			if !createIfNotExist {
				return 0, "User account has not been created yet. A site admin may have to create one for you.", err
			}
			// user was not found and we should create a new one
			return createUser(ctx, newOrUpdatedUser, externalAccount, data)
		}
	}

	// Update user in our DB if their profile info changed on the issuer. (Except username,
	// which the user is somewhat likely to want to control separately on Sourcegraph.)
	user, err := db.Users.GetByID(ctx, userID)
	if err != nil {
		return 0, "Unexpected error getting the Sourcegraph user account. Ask a site admin for help.", err
	}
	var userUpdate db.UserUpdate
	if user.DisplayName != newOrUpdatedUser.DisplayName {
		userUpdate.DisplayName = &newOrUpdatedUser.DisplayName
	}
	if user.AvatarURL != newOrUpdatedUser.AvatarURL {
		userUpdate.AvatarURL = &newOrUpdatedUser.AvatarURL
	}
	if userUpdate != (db.UserUpdate{}) {
		if err := db.Users.Update(ctx, user.ID, userUpdate); err != nil {
			return 0, "Unexpected error updating the Sourcegraph user account with new user profile information from the external account. Ask a site admin for help.", err
		}
	}
	return user.ID, "", nil
}

func createUser(ctx context.Context, newUser db.NewUser, externalAccount extsvc.ExternalAccountSpec, data extsvc.ExternalAccountData) (
	userID int32, safeErrMsg string, err error,
) {
	// Looser requirements: if the external auth provider returns a username or email that
	// already exists, just use that user instead of refusing.
	const allowMatchOnUsernameOrEmailOnly = true
	associateUser := false

	userID, err = db.ExternalAccounts.CreateUserAndSave(ctx, newUser, externalAccount, data)
	switch {
	case db.IsUsernameExists(err):
		if allowMatchOnUsernameOrEmailOnly {
			user, err2 := db.Users.GetByUsername(ctx, newUser.Username)
			if err2 == nil {
				userID = user.ID
				err = nil
				associateUser = true
			} else {
				log15.Error("Unable to reuse user account with username for authentication via external provider.", "username", newUser.Username, "err", err)
			}
		}
		safeErrMsg = fmt.Sprintf("The Sourcegraph username %q already exists and is not linked to this external account. If possible, sign using the external account you used previously. If that's not possible, a site admin can unlink or delete the Sourcegraph account with that username to fix this problem.", newUser.Username)
	case db.IsEmailExists(err):
		if allowMatchOnUsernameOrEmailOnly {
			user, err2 := db.Users.GetByVerifiedEmail(ctx, newUser.Email)
			if err2 == nil {
				userID = user.ID
				err = nil
				associateUser = true
			} else {
				log15.Error("Unable to reuse user account with email for authentication via external provider.", "email", newUser.Email, "err", err)
			}
		}
		safeErrMsg = fmt.Sprintf("The email address %q already exists and is associated with a different Sourcegraph user. A site admin can remove the email address from that Sourcegraph user to fix this problem.", newUser.Email)
	case errcode.PresentationMessage(err) != "":
		safeErrMsg = errcode.PresentationMessage(err)
	case err != nil:
		safeErrMsg = "Unable to create a new user account due to a conflict or other unexpected error. Ask a site admin for help."
	}

	if associateUser {
		if err := db.ExternalAccounts.AssociateUserAndSave(ctx, userID, externalAccount, data); err != nil {
			safeErrMsg = "Unexpected error associating the external account with the existing Sourcegraph user with the same username or email address."
			return 0, safeErrMsg, err
		}
	}

	return userID, safeErrMsg, err
}
