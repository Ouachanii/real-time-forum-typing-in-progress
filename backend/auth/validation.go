package auth

import (
	"fmt"
	"log"
	"regexp"
	"unicode"
)

func isLetter(s string) bool {
	for _, char := range s {
		if !unicode.IsLetter(char) {
			return false
		}
	}
	return true
}

func validateNickname(nickname string) string {
	const maxNickname = 50
	if len(nickname) == 0 {
		return "Nickname cannot be empty"
	} else if len(nickname) > maxNickname {
		return fmt.Sprintf("Nickname cannot be longer than %d characters", maxNickname)
	} else if !isLetter(nickname) {
		return "Nickname should combined with just letters (a-z) (A-Z)"
	}
	return ""
}

func validateEmail(email string) string {
	const maxEmail = 100
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if len(email) == 0 {
		return "Email cannot be empty"
	} else if len(email) > maxEmail {
		return fmt.Sprintf("Email cannot be longer than %d characters", maxEmail)
	} else if !emailRegex.MatchString(email) {
		return "Invalid email format"
	}
	return ""
}

func validatePassword(password string) string {
	const maxPassword = 100
	if len(password) < 8 {
		return "Password must be at least 8 characters long"
	} else if len(password) > maxPassword {
		return fmt.Sprintf("Password cannot be longer than %d characters", maxPassword)
	} else {
		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		hasDigit := regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial := regexp.MustCompile(`[\W_]`).MatchString(password)

		if !hasUpper {
			return "Password must include at least one uppercase letter"
		}
		if !hasLower {
			return "Password must include at least one lowercase letter"
		}
		if !hasDigit {
			return "Password must include at least one digit"
		}
		if !hasSpecial {
			return "Password must include at least one special character"
		}
	}
	return ""
}

func validateName(name string, fieldName string, maxLen int) string {
	if len(name) == 0 {
		return fmt.Sprintf("%s cannot be empty", fieldName)
	} else if len(name) > maxLen {
		return fmt.Sprintf("%s cannot be longer than %d characters", fieldName, maxLen)
	}
	return ""
}

func Validation(nickname, email, password, firstName, lastName string, age int, gender string) (map[string]string, bool) {
	errors := make(map[string]string)
	const maxFirstName = 50
	const maxLastName = 50

	// Nickname validation
	if err := validateNickname(nickname); err != "" {
		errors["nickname"] = err
	}

	// Email validation
	if err := validateEmail(email); err != "" {
		errors["email"] = err
	}

	// Password validation
	if err := validatePassword(password); err != "" {
		errors["password"] = err
	}

	// Name validation
	if err := validateName(firstName, "First name", maxFirstName); err != "" {
		errors["first_name"] = err
	}

	if err := validateName(lastName, "Last name", maxLastName); err != "" {
		errors["last_name"] = err
	}

	// Age validation
	if age < 18 || age >= 150 {
		errors["age"] = "Age must be between 18 and 150"
	}

	// Gender validation
	if gender != "Male" && gender != "Female" {
		errors["gender"] = "Gender must be either 'Male' or 'Female'"
	}

	if len(errors) > 0 {
		log.Println("Validation errors:", errors)
		return errors, false
	}
	return nil, true
}
