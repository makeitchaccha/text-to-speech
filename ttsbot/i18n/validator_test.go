package i18n

import (
	"testing"
)

type (
	resourceA struct {
		Field1 string
		Field2 string
	}
	resourceB struct {
		Field1 string
		Field2 string
		Field3 resourceA
	}
)

type testCase struct {
	Name           string
	Resource       interface{}
	ExpectedErrors int
}

func TestVerifyCompleteness(t *testing.T) {
	validCases := []testCase{
		{
			Name: "Valid Resource A",
			Resource: resourceA{
				Field1: "value1",
				Field2: "value2",
			},
			ExpectedErrors: 0,
		},
		{
			Name: "Valid Resource B",
			Resource: resourceB{
				Field1: "value1",
				Field2: "value2",
				Field3: resourceA{
					Field1: "subvalue1",
					Field2: "subvalue2",
				},
			},
			ExpectedErrors: 0,
		},
	}

	for _, res := range validCases {
		t.Run(res.Name, func(t *testing.T) {
			errs := verifyCompleteness(res.Resource, "Root")
			if len(errs) != res.ExpectedErrors {
				t.Errorf("len(errs) = %d errors, expected %d", len(errs), res.ExpectedErrors)
			}
		})
	}

	invalidCases := []testCase{
		{
			Name: "Invalid Resource A Empty Field1",
			Resource: resourceA{
				Field1: "",
				Field2: "value2",
			},
			ExpectedErrors: 1,
		},
		{
			Name: "Invalid Resource B Empty SubField2",
			Resource: resourceB{
				Field1: "value1",
				Field2: "value2",
				Field3: resourceA{
					Field1: "subvalue1",
					Field2: "",
				},
			},
			ExpectedErrors: 1,
		},
		{
			Name: "Invalid Resource B All Fields Empty",
			Resource: resourceB{
				Field1: "",
				Field2: "",
				Field3: resourceA{
					Field1: "",
					Field2: "",
				},
			},
			ExpectedErrors: 4,
		},
	}

	for _, res := range invalidCases {
		t.Run(res.Name, func(t *testing.T) {
			errs := verifyCompleteness(res.Resource, "Root")
			if len(errs) != res.ExpectedErrors {
				t.Errorf("len(errs) = %d errors, expected %d", len(errs), res.ExpectedErrors)
			}
		})
	}
}
