package services

import (
	"errors"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/verify/v2"
)

type TwilioConfig struct {
	verifyServiceSid   string
	customMessage      string
	customFriendlyName string
	client             *twilio.RestClient
}

func NewOTPService(twilioAccountSid string, verifyServiceSid string, twilioAuthToken string,
	customMessage string, customFriendlyName string) *TwilioConfig {
	return &TwilioConfig{
		verifyServiceSid:   verifyServiceSid,
		customMessage:      customMessage,
		customFriendlyName: customFriendlyName,
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: twilioAccountSid,
			Password: twilioAuthToken,
		}),
	}
}

func (tc *TwilioConfig) SendOTP(phonenumber string) (bool, error) {
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

func (tc *TwilioConfig) VerifyOTP(phonenumber string, code string) (bool, error) {
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
