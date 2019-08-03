// Copyright 2019 Keybase, Inc. All rights reserved. Use of
// this source code is governed by the included BSD license.

package systests

import (
	"testing"

	"github.com/keybase/client/go/contacts"
	"github.com/keybase/client/go/kbtest"
	"github.com/keybase/client/go/libkb"
	"github.com/keybase/client/go/protocol/keybase1"

	"github.com/stretchr/testify/require"
	context "golang.org/x/net/context"
)

func TestLookupContactList(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	ann := tt.addUser("ann")

	phone := keybase1.PhoneNumber("+" + kbtest.GenerateTestPhoneNumber())

	phoneCli := keybase1.PhoneNumbersClient{Cli: ann.teamsClient.Cli}
	err := phoneCli.AddPhoneNumber(context.Background(), keybase1.AddPhoneNumberArg{
		PhoneNumber: phone,
		Visibility:  keybase1.IdentityVisibility_PUBLIC,
	})
	require.NoError(t, err)

	code, err := kbtest.GetPhoneVerificationCode(ann.MetaContext(), phone)
	require.NoError(t, err)

	err = phoneCli.VerifyPhoneNumber(context.Background(), keybase1.VerifyPhoneNumberArg{
		PhoneNumber: phone,
		Code:        code,
	})
	require.NoError(t, err)

	contactsCli := keybase1.ContactsClient{Cli: ann.teamsClient.Cli}
	rawPhone := keybase1.RawPhoneNumber(phone)
	res, err := contactsCli.LookupContactList(context.Background(), keybase1.LookupContactListArg{
		Contacts: []keybase1.Contact{
			keybase1.Contact{Name: "It's me",
				Components: []keybase1.ContactComponent{
					keybase1.ContactComponent{
						PhoneNumber: &rawPhone,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	contactRes := res[0]
	require.True(t, contactRes.Resolved)
	require.Equal(t, ann.uid, contactRes.Uid)
	require.NotNil(t, contactRes.Component.PhoneNumber)
	require.Equal(t, rawPhone, *contactRes.Component.PhoneNumber)

	mctx := libkb.NewMetaContextForTest(*ann.tc)
	emailAddr := keybase1.EmailAddress(ann.userInfo.email)
	err = kbtest.VerifyEmailAuto(mctx, emailAddr)
	require.NoError(t, err)

	emailCli := keybase1.EmailsClient{Cli: ann.teamsClient.Cli}
	err = emailCli.SetVisibilityEmail(context.Background(), keybase1.SetVisibilityEmailArg{
		Email:      emailAddr,
		Visibility: keybase1.IdentityVisibility_PUBLIC,
	})
	require.NoError(t, err)

	res, err = contactsCli.LookupContactList(context.Background(), keybase1.LookupContactListArg{
		Contacts: []keybase1.Contact{
			keybase1.Contact{Name: "It's me",
				Components: []keybase1.ContactComponent{
					keybase1.ContactComponent{
						Email: &emailAddr,
					},
				},
			},
		},
	})
	require.NoError(t, err)
	require.Len(t, res, 1)
	contactRes = res[0]
	require.True(t, contactRes.Resolved)
	require.Equal(t, ann.uid, contactRes.Uid)
	require.NotNil(t, contactRes.Component.Email)
	require.Equal(t, emailAddr, *contactRes.Component.Email)
}

func TestBulkLookupContacts(t *testing.T) {
	tt := newTeamTester(t)
	defer tt.cleanup()

	// User looking up other contacts
	actor := tt.addUser("blac")
	actorCtx := libkb.NewMetaContextForTest(*actor.tc)

	// Some helper functions because we're preparing a few accounts at once
	addEmailAddress := func(user *userPlusDevice, vis keybase1.IdentityVisibility, verify bool) keybase1.EmailAddress {
		emailAddr := keybase1.EmailAddress(user.userInfo.email)

		mctx := libkb.NewMetaContextForTest(*user.tc)
		if verify {
			err := kbtest.VerifyEmailAuto(mctx, emailAddr)
			require.NoError(t, err)
		}

		if vis == keybase1.IdentityVisibility_PUBLIC {
			emailCli := keybase1.EmailsClient{Cli: user.teamsClient.Cli}
			err := emailCli.SetVisibilityEmail(context.Background(), keybase1.SetVisibilityEmailArg{
				Email:      emailAddr,
				Visibility: keybase1.IdentityVisibility_PUBLIC,
			})
			require.NoError(t, err)
		}

		return emailAddr
	}
	addPhoneNumber := func(user *userPlusDevice, vis keybase1.IdentityVisibility, verify bool) keybase1.PhoneNumber {
		phone := keybase1.PhoneNumber("+" + kbtest.GenerateTestPhoneNumber())
		phoneCli := keybase1.PhoneNumbersClient{Cli: user.teamsClient.Cli}
		err := phoneCli.AddPhoneNumber(context.Background(), keybase1.AddPhoneNumberArg{
			PhoneNumber: phone,
			Visibility:  vis,
		})
		require.NoError(t, err)

		if verify {
			code, err := kbtest.GetPhoneVerificationCode(user.MetaContext(), phone)
			require.NoError(t, err)

			err = phoneCli.VerifyPhoneNumber(context.Background(), keybase1.VerifyPhoneNumberArg{
				PhoneNumber: phone,
				Code:        code,
			})
			require.NoError(t, err)
		}

		return phone
	}

	// Someone with a verified visible email #1
	evv1 := tt.addUser("blevv")
	evv1Email := addEmailAddress(evv1, keybase1.IdentityVisibility_PUBLIC, true)
	// Someone with a verified visible email #2
	evv2 := tt.addUser("blevv")
	evv2Email := addEmailAddress(evv2, keybase1.IdentityVisibility_PUBLIC, true)
	// Someone with a verified private email
	evp := tt.addUser("blevp")
	evpEmail := addEmailAddress(evp, keybase1.IdentityVisibility_PRIVATE, true)
	// Someone with an unverified visible email
	epv := tt.addUser("blepv")
	epvEmail := addEmailAddress(epv, keybase1.IdentityVisibility_PUBLIC, false)

	// Someone with a verified visible phone number #1
	pvv1 := tt.addUser("blpvv")
	pvv1Number := addPhoneNumber(pvv1, keybase1.IdentityVisibility_PUBLIC, true)
	pvv1NumberRaw := keybase1.RawPhoneNumber(pvv1Number)
	// Someone with a verified visible phone number #2, without prefix
	pvv2 := tt.addUser("blpvv")
	pvv2Number := addPhoneNumber(pvv2, keybase1.IdentityVisibility_PUBLIC, true)
	pvv2NumberRaw := keybase1.RawPhoneNumber(pvv2Number[2:])
	// Someone with a verified private phone number
	pvp := tt.addUser("blpvp")
	pvpNumber := addPhoneNumber(pvp, keybase1.IdentityVisibility_PRIVATE, false)
	pvpNumberRaw := keybase1.RawPhoneNumber(pvpNumber)
	// Someone with an unverified visible phone number
	ppv := tt.addUser("blppv")
	ppvNumber := addPhoneNumber(ppv, keybase1.IdentityVisibility_PUBLIC, false)
	ppvNumberRaw := keybase1.RawPhoneNumber(ppvNumber)

	// A few unused numbers
	const randomEmail = "doesnotexist@example.org"
	const randomNumber = "+12025550116"

	// Run the lookup
	res, err := contacts.BulkLookupContacts(
		actorCtx,
		[]keybase1.EmailAddress{
			evv1Email,
			evv2Email,
			evpEmail,
			epvEmail,
			randomEmail,
		},
		[]keybase1.RawPhoneNumber{
			pvv1NumberRaw,
			pvv2NumberRaw,
			pvpNumberRaw,
			ppvNumberRaw,
			randomNumber,
		},
		keybase1.RegionCode("US"),
	)
	require.NoError(t, err)

tableLoop:
	for _, x := range []struct {
		LookupKey contacts.ContactLookupKey
		Match     bool
		Coerced   bool
	}{
		{contacts.MakeEmailLookupKey(evv1Email), true, false},
		{contacts.MakeEmailLookupKey(evv2Email), true, false},
		{contacts.MakeEmailLookupKey(evpEmail), false, false},
		{contacts.MakeEmailLookupKey(epvEmail), false, false},
		{contacts.MakeEmailLookupKey(randomEmail), false, false},
		{contacts.MakePhoneLookupKey(pvv1NumberRaw), true, false},
		{contacts.MakePhoneLookupKey(pvv2NumberRaw), true, true},
		{contacts.MakePhoneLookupKey(pvpNumberRaw), false, false},
		{contacts.MakePhoneLookupKey(ppvNumberRaw), false, false},
		{contacts.MakeEmailLookupKey(randomNumber), false, false},
	} {
		for k, v := range res.Results {
			if k != x.LookupKey {
				continue
			}

			// We found one!
			if !x.Match {
				require.Fail(t, "found %v in the result", x.LookupKey)
				continue tableLoop
			}

			// Evaluate coerced
			require.True(
				t,
				(x.Coerced && v.Coerced != "") ||
					(!x.Coerced && v.Coerced == ""),
				"%v coerced value was expected to be %v, got %v",
				x.LookupKey, x.Coerced, v.Coerced != "",
			)
			continue tableLoop
		}

		// We didn't find anything
		if x.Match {
			require.Fail(t, "did not find %v in the result", x.LookupKey)
		}
	}
}
