package utility

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

func PasswordValidator(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// checking for atleast one uppercase letter
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)

	// checking for atleast one lowercase letter
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)

	// checking for atleast one number
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)

	// checking for atleat one special character
	hasSpecialCharacter := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecialCharacter
}

func UsernameAndGroupnameValidator(fl validator.FieldLevel) bool {
	field := fl.Field().String()
	return regexp.MustCompile(`^[a-zA-Z0-9]_`).MatchString(field)
}

func PhonenumberValidator(fl validator.FieldLevel) bool {
	phonenumber := fl.Field().String()
	phonenumberRegex := regexp.MustCompile(`^(?:(?:\+91|0)?[ -]?)?(?:(?:\d{2,4}[ -]?\d{6,8})|(?:\d{10}))$`)
	return phonenumberRegex.MatchString(phonenumber)
}
