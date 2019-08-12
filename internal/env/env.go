package env

import "strings"

// EnvType type alias
type Type string

// DEV environment
const DEV Type = "dev"

// PROD environment
const PROD Type = "prod"

// QA environment
const QA Type = "qa"

// testing environment
const TEST Type = "test"

var validTypes = []Type{
	DEV,
	PROD,
	QA,
	TEST,
}

func ValidTypeOrDev(typ Type) Type {
	if ValidType(typ) {
		return Type(typ)
	}

	return DEV
}

func ValidType(typ Type) bool {
	for _, t := range validTypes {
		if strings.ToLower(string(typ)) == strings.ToLower(string(t)) {
			return true
		}
	}
	return false
}

func (e Type) IsProd() bool {
	return strings.Contains(string(e), string(PROD))
}
func (e Type) IsQA() bool {
	return strings.Contains(string(e), string(QA))
}
func (e Type) IsTest() bool {
	return strings.Contains(string(e), string(TEST))
}
