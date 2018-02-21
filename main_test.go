package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitTo(t *testing.T) {
	a := []string{
		"Hello World",
		"Hello Universe",
		"Good Bye",
	}
	b := limitTo(a, 6)
	require.EqualValues(t, []string{
		"Hello ",
		"World",
		"Hello ",
		"Univer",
		"se",
		"Good B",
		"ye",
	}, b)
}
