package utils

import uuid "github.com/satori/go.uuid"

func UUID() string {
	return uuid.NewV4().String()
}

func ValidUUIDString(sid string) bool {
	_, err := uuid.FromString(sid)
	return err == nil
}
