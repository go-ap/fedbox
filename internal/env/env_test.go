package env

import "testing"

func TestType_IsProd(t *testing.T) {
	prod := PROD
	if !prod.IsProd() {
		t.Errorf("%T %s should have been production", prod, prod)
	}
	qa := QA
	if qa.IsProd() {
		t.Errorf("%T %s should not have been production", qa, qa)
	}
	dev := DEV
	if dev.IsProd() {
		t.Errorf("%T %s should not have been production", dev, dev)
	}
	test := TEST
	if test.IsProd() {
		t.Errorf("%T %s should not have been production", test, test)
	}
	rand := Type("Random")
	if rand.IsProd() {
		t.Errorf("%T %s should not have been production", rand, rand)
	}
}

func TestType_IsQA(t *testing.T) {
	qa := QA
	if !qa.IsQA() {
		t.Errorf("%T %s should not have been qa", qa, qa)
	}
	prod := PROD
	if prod.IsQA() {
		t.Errorf("%T %s should not have been qa", prod, prod)
	}
	dev := DEV
	if dev.IsQA() {
		t.Errorf("%T %s should not have been qa", dev, dev)
	}
	test := TEST
	if test.IsQA() {
		t.Errorf("%T %s shouldhave been qa", test, test)
	}
	rand := Type("Random")
	if rand.IsQA() {
		t.Errorf("%T %s should not have been qa", rand, rand)
	}
}

func TestType_IsTest(t *testing.T) {
	test := TEST
	if !test.IsTest() {
		t.Errorf("%T %s should have been test", test, test)
	}
	prod := PROD
	if prod.IsTest() {
		t.Errorf("%T %s should not have been test", prod, prod)
	}
	qa := QA
	if qa.IsTest() {
		t.Errorf("%T %s should not have been test", qa, qa)
	}
	dev := DEV
	if dev.IsTest() {
		t.Errorf("%T %s should not have been test", dev, dev)
	}
	rand := Type("Random")
	if rand.IsTest() {
		t.Errorf("%T %s should not have been test", rand, rand)
	}
}

func TestValidTypeOrDev(t *testing.T) {
	prod := PROD
	if prod != ValidTypeOrDev(string(prod)) {
		t.Errorf("%T %s should have been valid, received %s", prod, prod, ValidTypeOrDev(string(prod)))
	}
	qa := QA
	if qa != ValidTypeOrDev(string(qa)) {
		t.Errorf("%T %s should have been valid, received %s", qa, qa, ValidTypeOrDev(string(qa)))
	}
	test := TEST
	if test != ValidTypeOrDev(string(test)) {
		t.Errorf("%T %s should have been valid, received %s", test, test, ValidTypeOrDev(string(test)))
	}
	dev := DEV
	if dev != ValidTypeOrDev(string(dev)) {
		t.Errorf("%T %s should have been valid, received %s", dev, dev, ValidTypeOrDev(string(dev)))
	}
	rand := "Random"
	if dev != ValidTypeOrDev(rand) {
		t.Errorf("%T %s should not have been valid, received %s", rand, rand, ValidTypeOrDev(rand))
	}
}

func TestValidType(t *testing.T) {
	prod := PROD
	if !ValidType(string(prod)) {
		t.Errorf("%T %s should have been valid", prod, prod)
	}
	qa := QA
	if !ValidType(string(qa)) {
		t.Errorf("%T %s should have been valid", qa, qa)
	}
	dev := DEV
	if! ValidType(string(dev)) {
		t.Errorf("%T %s should have been valid", dev, dev)
	}
	test := TEST
	if !ValidType(string(test)) {
		t.Errorf("%T %s should have been valid", test, test)
	}
	rand := "Random"
	if ValidType(rand) {
		t.Errorf("%T %s should not have been valid", rand, rand)
	}
}
