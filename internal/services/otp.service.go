package services

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/verify/v2"
)

const (
	expiresAfter = 10 // duration for which OTP is valid after this it becomes invalid or expired
)

// this struct is used to cache the expiration time for the otp sent to user phonenumber
type otpCache struct {
	otpcache map[string]*time.Time // key: phonenumber of user, value: time at which otp will expire
	mutex    sync.RWMutex
	stop     chan struct{} // channel to recieve signal to stop monitoring the cache
}

type TwilioConfig struct {
	verifyServiceSid string
	channel          string
	client           *twilio.RestClient
	otpcache         *otpCache
}

/*
monitorAndRemove function will be used to check for stale cache entries in otpCache
and remove them as they no longer can get verified because they are expired

the duration for monitoring will be 10min i.e. every 10min this function will check
whether there are stale entries or not and if found any then they will be removed
from cache
*/
func (oc *otpCache) monitorAndRemove() {
	monitoringInterval := time.NewTicker(time.Minute * expiresAfter)
	select {
	case <-monitoringInterval.C:
		// call the function to check and remove stale entries from cache
		oc.checkAndRemove()
	case <-oc.stop:
		return
	}
}

/*
checkAndRemove function will be called every 10mins to from monitorAndRemove to check
and remove any stale entries present in the cache
*/
func (oc *otpCache) checkAndRemove() {
	// acquiring the lock
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	// checking and removing stale entries
	for key, value := range oc.otpcache {
		if time.Now().Equal(*value) || time.Now().After(*value) {
			delete(oc.otpcache, key)
		}
	}
}

/*
set function belongs to otpCache struct which will add a new entry in the cache

@param phonenumber: user phonenumber on which otp was requested to uniquely
identify the expiration time of the otp sent to user

@param value: pointer to time.Time struct which contains the expiration time of otp
*/
func (oc *otpCache) set(phonenumber string, value *time.Time) {
	// acquiring the lock
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	oc.otpcache[phonenumber] = value
}

/*
get function belongs to otpCache struct which will return only valid entry's from the cache.
When a request comes to this function it will first check if the requested cache value has
expired or not then based on the result it will return either (nil, err) or (value, nil)

@param phonenumber: user phonenumber on which otp was requested
*/
func (oc *otpCache) get(phonenumber string) (*time.Time, error) {
	// acquiring the read lock
	oc.mutex.RLock()
	defer oc.mutex.RUnlock()

	value, ok := oc.otpcache[phonenumber]
	if !ok {
		return nil, fmt.Errorf("no entry found for phonenumber: %s", phonenumber)
	}

	return value, nil
}

/*
remove function belongs to otpCache struct which will be used to remove a stale value or the value
for which OTP has been successfully verified.

@param phonenumber: phonenumber of the user on which otp is requested
*/
func (oc *otpCache) remove(phonenumber string) error {
	// acquiring the lock
	oc.mutex.Lock()
	defer oc.mutex.Unlock()

	if _, ok := oc.otpcache[phonenumber]; !ok {
		return fmt.Errorf("no entry found for verification token %s to remove", phonenumber)
	}

	delete(oc.otpcache, phonenumber)
	return nil
}

func NewOTPService(twilioAccountSid, verifyServiceSid, twilioAuthToken, channel string) *TwilioConfig {
	log.Printf("[OTP_SERVICE]: started otp service")
	twilioConfig := &TwilioConfig{
		verifyServiceSid: verifyServiceSid,
		channel:          channel,
		client: twilio.NewRestClientWithParams(twilio.ClientParams{
			Username: twilioAccountSid,
			Password: twilioAuthToken,
		}),
		otpcache: &otpCache{
			otpcache: make(map[string]*time.Time),
			stop:     make(chan struct{}),
		},
	}

	go twilioConfig.otpcache.monitorAndRemove()
	return twilioConfig
}

func (tc *TwilioConfig) SendOTP(phonenumber string) error {
	// checking if there is already an non-expired otp sent to user
	// if the otp is present and expired then new otp will be sent
	// otherwise the duration after which the user can request otp will be sent as response
	expirationTime, err := tc.otpcache.get(phonenumber)
	if err != nil || time.Now().Equal(*expirationTime) {
		params := &openapi.CreateVerificationParams{
			To:      &phonenumber,
			Channel: &tc.channel,
		}

		response, err := tc.client.VerifyV2.CreateVerification(tc.verifyServiceSid, params)
		if err != nil {
			log.Printf("[OTP_SERVICE]: Error sending otp to user %v", err)
			return err
		}

		// setting new entry in the otpcache
		tc.otpcache.set(phonenumber, response.DateCreated)
		return nil
	}

	// returning the duration after which user is allowed to request for new otp
	return fmt.Errorf("you are allowed to request for new otp after %s", time.Until(*expirationTime))
}

func (tc *TwilioConfig) VerifyOTP(phonenumber string, code string) error {
	// if the user's otp is successfully verified then its entry from otpcache will be removed
	// if it fails then before returning check if otp is expired then remove it from cache
	// if response status is expired then also remove it from cache
	params := &openapi.CreateVerificationCheckParams{}
	params.SetTo(phonenumber)
	params.SetCode(code)

	response, err := tc.client.VerifyV2.CreateVerificationCheck(tc.verifyServiceSid, params)
	if err != nil {
		log.Printf("[OTP_SERVICE]: Error requesting twilio api for sending otp to user phonenumber %s, %v", phonenumber, err)
		return err
	}

	switch *response.Status {
	case "approved":
		// removing otp entry from cache
		if err = tc.otpcache.remove(phonenumber); err != nil {
			log.Printf("[OTP_SERVICE]: Error removing otp entry from otpcache after successfull approval %v", err)
			return err
		}

		return nil
	case "failed":
		// checking if otp is expired then removing it from cache
		expirationTime, err := tc.otpcache.get(phonenumber)
		if err != nil {
			log.Printf("[OTP_SERVICE]: Error fetching the otp entry from otpcache after failed approval %v", err)
			return err
		}

		if time.Now().Equal(*expirationTime) {
			if err = tc.otpcache.remove(phonenumber); err != nil {
				log.Printf("[OTP_SERVICE]: Error removing otp entry from otpcache after failed approval %v", err)
				return err
			}
		}
		return errors.New("incorrect otp")
	case "expired":
		if err = tc.otpcache.remove(phonenumber); err != nil {
			log.Printf("[OTP_CACHE]: Error removing otp entry from otpcache after it is expired %v", err)
			return err
		}
		return errors.New("otp expired")
	}

	return nil
}
