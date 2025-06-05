package services

import (
	"errors"
	"log"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/verify/v2"
)

type TwilioConfig struct {
	verifyServiceSid string
	channel          string
	client           *twilio.RestClient
}

func NewOTPService(twilioAccountSid, verifyServiceSid, twilioAuthToken, channel string) *TwilioConfig {
	log.Printf("[OTP_SERVICE]: started otp service")
	return &TwilioConfig{
		verifyServiceSid: verifyServiceSid,
		channel:          channel,
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: twilioAccountSid,
			Password: twilioAuthToken,
		}),
	}
}

func (tc *TwilioConfig) SendOTP(phonenumber string) error {
	params := &openapi.CreateVerificationParams{
		To:      &phonenumber,
		Channel: &tc.channel,
	}

	response, err := tc.client.VerifyV2.CreateVerification(tc.verifyServiceSid, params)
	log.Println(response.SendCodeAttempts)
	if err != nil {
		return err
	}

	return nil
}

func (tc *TwilioConfig) VerifyOTP(phonenumber string, code string) error {
	params := &openapi.CreateVerificationCheckParams{}
	params.SetTo(phonenumber)
	params.SetCode(code)

	response, err := tc.client.VerifyV2.CreateVerificationCheck(tc.verifyServiceSid, params)
	if err != nil {
		return err
	}

	switch *response.Status {
	case "approved":
		return nil
	case "failed":
		return errors.New("incorrect otp")
	case "expired":
		return errors.New("otp expired")
	}

	return nil
}
