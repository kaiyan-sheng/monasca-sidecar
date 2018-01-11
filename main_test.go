// (C) Copyright 2017-2018 Hewlett Packard Enterprise Development LP

package main

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestParseFloat(t *testing.T) {
	string1 := "30"
	float1, err1 := strconv.ParseFloat(string1, 64)
	assert.Equal(t, 30.0, float1)
	assert.NoError(t, err1)

	string2 := "30.0"
	float2, err2 := strconv.ParseFloat(string2, 64)
	assert.Equal(t, 30.0, float2)
	assert.NoError(t, err2)

	string3 := "not a float"
	_, err3 := strconv.ParseFloat(string3, 64)
	assert.Error(t, err3)

	string4 := "0"
	_, err4 := strconv.ParseFloat(string4, 64)
	assert.NoError(t, err4)

	string5 := "-30"
	float5, err5 := strconv.ParseFloat(string5, 64)
	assert.Equal(t, -30.0, float5)
	assert.NoError(t, err5)
}
