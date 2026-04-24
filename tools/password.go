package tools

import "golang.org/x/crypto/bcrypt"

func HashPassword(password string) (string, error) {
	hashedPassword, hashError := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if hashError != nil {
		return "", hashError
	}
	return string(hashedPassword), nil
}

func VerifyPassword(hashedPassword string, password string) bool {
	if hashedPassword == "" || password == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password)) == nil
}
