// Code generated by datamapper.
// https://github.com/underbek/datamapper

// Package with_filed_pointers is a generated datamapper package.
package with_filed_pointers

import "fmt"

// ConvertFromToTo convert From by tag map to To by tag map
func ConvertFromToTo(from From) (To, error) {
	if from.Age == nil {
		return To{}, fmt.Errorf("cannot convert From.Age -> To.Age, field is nil")
	}

	return To{
		UUID: &from.ID,
		Name: from.Name,
		Age:  *from.Age,
	}, nil
}
