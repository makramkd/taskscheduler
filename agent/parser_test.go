package agent

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type parseTestCase struct {
	input            string
	expectedDuration time.Duration
}

func TestParseFrequency(t *testing.T) {
	testCases := []parseTestCase{
		{
			input:            "every 15 minutes",
			expectedDuration: 15 * time.Minute,
		},
		{
			input:            "every 30 hours",
			expectedDuration: 30 * time.Hour,
		},
		{
			input:            "every 66 seconds",
			expectedDuration: 66 * time.Second,
		},
	}
	for _, testCase := range testCases {
		actualDuration, err := parseFrequency(testCase.input)
		assert.NoError(t, err)
		assert.Equal(t, testCase.expectedDuration, actualDuration)
	}
}

func TestParseFrequencyErrors(t *testing.T) {
	_, err := parseFrequency("kindof every 15 minutes")
	assert.Error(t, err)

	_, err = parseFrequency("haha 15 minutes")
	assert.Error(t, err)

	_, err = parseFrequency("every fifteen minutes")
	assert.Error(t, err)

	_, err = parseFrequency("every 2 days")
	assert.Error(t, err)
}
