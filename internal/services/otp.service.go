package service

import (
	"errors"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/verify/v2"
)

type twilioConfig struct {
	twilioAccountSid   string
	verifyServiceSid   string
	twilioAuthToken    string
	customMessage      string
	customFriendlyName string
	client             *twilio.RestClient
}

func NewOTPCache(twilioAccountSid string, verifyServiceSid string, twilioAuthToken string,
	customMessage string, customFriendlyName string, client *twilio.RestClient) *twilioConfig {
	return &twilioConfig{
		twilioAccountSid:   twilioAccountSid,
		verifyServiceSid:   verifyServiceSid,
		twilioAuthToken:    twilioAuthToken,
		customMessage:      customMessage,
		customFriendlyName: customFriendlyName,
		client:             client,
	}
}

func (tc *twilioConfig) SendOTP(phonenumber string) (bool, error) {
	params := &openapi.CreateVerificationParams{}
	params.SetTo(phonenumber)
	params.SetChannel("sms")
	params.SetCustomFriendlyName(tc.customFriendlyName)
	params.SetCustomMessage(tc.customMessage)

	_, err := tc.client.VerifyV2.CreateVerification(tc.verifyServiceSid, params)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (tc *twilioConfig) VerifyOTP(phonenumber string, code string) (bool, error) {
	params := &openapi.CreateVerificationCheckParams{}
	params.SetTo(phonenumber)
	params.SetCode(code)

	response, err := tc.client.VerifyV2.CreateVerificationCheck(tc.verifyServiceSid, params)
	if err != nil {
		return false, err
	}

	switch *response.Status {
	case "approved":
		return true, nil
	case "failed":
		return false, errors.New("incorrect otp")
	case "expired":
		return false, errors.New("otp expired")
	}

	return false, nil
}
