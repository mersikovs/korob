package models_test

import (
	"fmt"

	models "github.com/mersikovs/korob.git/internal/model"
)

// ExampleValidateMetricType_valid demonstrates validation of a known metric type.
func ExampleValidateMetricType_valid() {
	fmt.Println(models.ValidateMetricType(models.Gauge))

	// Output:
	// true
}

// ExampleValidateMetricType_invalid demonstrates validation of an unknown metric type.
func ExampleValidateMetricType_invalid() {
	fmt.Println(models.ValidateMetricType("unknown"))

	// Output:
	// false
}
