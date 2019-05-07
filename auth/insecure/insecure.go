package insecure

import "errors"

type InsecureAuth struct{}

func (a InsecureAuth) Key() string {
	return "X-INSECURE-AUTH"
}

func (a InsecureAuth) Verify(key, value string) (string, error) {
	if key == a.Key() && len(value) > 0 {
		return value, nil

	} else {
		return "", errors.New("Verification failed")
	}
}
