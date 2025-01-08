package main

import (
	"errors"
	"testing"
)

func TestHandlerLogin(t *testing.T) {
	cases := []struct {
		input    []string
		expected error
	}{
		{
			input:    []string{"login", "user", "password"},
			expected: errors.New("usage: login <username>"),
		},
		// { //I need to work out how to handle passing in a state object
		// 	input:    []string{"login", "captum"},
		// 	expected: nil,
		// },
	}

	for _, c := range cases {
		cmd := Command{Name: c.input[0], Args: c.input[1:]}
		err := handlerLogin(nil, cmd)
		if err.Error() != c.expected.Error() {
			t.Errorf("Expected %v but got %v", c.expected, err)
		}
	}
}
