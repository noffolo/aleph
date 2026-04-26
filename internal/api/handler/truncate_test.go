package handler

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTruncateJSON_ShortString(t *testing.T) {
	assert.Equal(t, "abc", truncateJSON("abc", 10))
	assert.Equal(t, "[1,2]", truncateJSON("[1,2]", 10))
}

func TestTruncateJSON_LongString(t *testing.T) {
	long := make([]byte, 100)
	for i := range long {
		long[i] = 'a'
	}
	result := truncateJSON(string(long), 5)
	assert.Equal(t, "aaaaa", result)
}

func TestTruncateJSON_ArrayWithObjects(t *testing.T) {
	arr := `[{"id":1,"data":"short"},{"id":2,"data":"very long string that exceeds limit"}]`
	result := truncateJSON(arr, 30)
	assert.True(t, len(result) <= len(arr), "truncated result should be shorter")
	assert.Contains(t, result, "...")
}

func TestTruncateJSON_SimpleObject(t *testing.T) {
	obj := `{"key1":"value1","key2":"value2","key3":"value3"}`
	result := truncateJSON(obj, 20)
	assert.Contains(t, result, "...")
}

func TestTruncateJSON_EmptyInput(t *testing.T) {
	assert.Equal(t, "", truncateJSON("", 10))
}

func TestTruncateJSON_ExactLimit(t *testing.T) {
	input := `"exact length"`
	assert.Equal(t, input, truncateJSON(input, len(input)))
}
