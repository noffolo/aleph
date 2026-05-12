//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	v1 "github.com/ff3300/aleph-v2/internal/api/proto/aleph/v1"
)

// Verifies that a valid protobuf message can be marshaled and unmarshaled.
func TestProtobufValidMessage(t *testing.T) {
	req := &v1.CreateProjectRequest{
		Id:   "proj-01",
		Name: "test-project",
	}

	data, err := proto.Marshal(req)
	require.NoError(t, err, "valid message should marshal without error")
	require.NotEmpty(t, data, "marshaled data should not be empty")

	var decoded v1.CreateProjectRequest
	err = proto.Unmarshal(data, &decoded)
	require.NoError(t, err, "valid binary should unmarshal without error")

	assert.Equal(t, "proj-01", decoded.GetId())
	assert.Equal(t, "test-project", decoded.GetName())
}

// Verifies that truncated/incomplete binary data is rejected.
func TestProtobufTruncatedMessage(t *testing.T) {
	req := &v1.ExecuteQueryRequest{
		ObjectType: "users",
		ProjectId:  "proj-01",
		Limit:      100,
	}

	data, err := proto.Marshal(req)
	require.NoError(t, err)

	// Truncate at various sizes and verify unmarshal fails
	for _, offset := range []int{1, len(data) / 4, len(data) / 2} {
		truncated := data[:offset]

		var decoded v1.ExecuteQueryRequest
		err = proto.Unmarshal(truncated, &decoded)
		assert.Error(t, err, "truncated data should fail unmarshal at offset %d", offset)
	}
}

// Verifies that oversized messages are handled sensibly.
// Tests with a large repeated string field to exercise wire-format size limits.
func TestProtobufOversizedMessage(t *testing.T) {
	// Build a ChatRequest with a very large message field
	largeMessage := strings.Repeat("abcdefghij", 1_000_000) // ~10MB

	req := &v1.ChatRequest{
		Message:   largeMessage,
		ProjectId: "proj-01",
		AgentId:   "agent-01",
	}

	data, err := proto.Marshal(req)
	// proto.Marshal may succeed or fail depending on internal limits.
	// The key assertion: if it succeeds, the round-trip must work.
	if err == nil {
		var decoded v1.ChatRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err, "round-trip of large message should succeed")
		assert.Equal(t, largeMessage, decoded.GetMessage())
	} else {
		// If marshal fails, the error should be meaningful
		assert.NotEmpty(t, err.Error(), "oversized marshal error should have a message")
	}
}

// Verifies that corrupt wire-format data is rejected.
func TestProtobufInvalidWireFormat(t *testing.T) {
	req := &v1.CreateProjectRequest{
		Id:   "proj-01",
		Name: "test",
	}

	validData, err := proto.Marshal(req)
	require.NoError(t, err)

	// Corrupt the wire format by flipping bits in the binary data
	corrupted := make([]byte, len(validData))
	copy(corrupted, validData)

	// Flip the first byte of each field-encoding section
	for i := range corrupted {
		if i >= 5 && corrupted[i] == 0x0a { // field tag for string (wire type 2)
			corrupted[i] = 0xff // invalid wire type
			break
		}
	}

	// If we can't find a 0x0a to corrupt past the first few bytes,
	// just corrupt high bits
	for i := range corrupted {
		if i >= 2 {
			corrupted[i] ^= 0x80
			break
		}
	}

	var decoded v1.CreateProjectRequest
	err = proto.Unmarshal(corrupted, &decoded)
	assert.Error(t, err, "corrupted wire format should fail unmarshal")
}

// Verifies that empty messages (zero-value) are valid proto3 messages.
func TestProtobufEmptyMessage(t *testing.T) {
	// An empty proto3 message is valid; all fields default to zero values
	req := &v1.CreateProjectRequest{}

	data, err := proto.Marshal(req)
	require.NoError(t, err)

	// Empty proto message with no set fields produces zero-length data
	assert.Equal(t, 0, len(data), "empty proto3 message should marshal to 0 bytes")

	// Unmarshaling an empty byte slice should produce a zero-value message
	var decoded v1.CreateProjectRequest
	err = proto.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, "", decoded.GetId())
	assert.Equal(t, "", decoded.GetName())
}

// Verifies wire-format integrity via round-trip marshal/unmarshal.
func TestProtobufWireFormatRoundTrip(t *testing.T) {
	// Test with various message types
	t.Run("CreateProjectRequest", func(t *testing.T) {
		original := &v1.CreateProjectRequest{
			Id:   "proj-test-42",
			Name: "integration-test-project",
		}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		var decoded v1.CreateProjectRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err)

		// Re-marshal the decoded message and compare
		dataAgain, err := proto.Marshal(&decoded)
		require.NoError(t, err)

		assert.Equal(t, data, dataAgain, "re-marshaled data should be identical")
		assert.True(t, proto.Equal(original, &decoded), "messages should be equal after round-trip")
	})

	t.Run("ExecuteQueryRequest", func(t *testing.T) {
		original := &v1.ExecuteQueryRequest{
			ObjectType: "orders",
			ProjectId:  "proj-biz",
			Limit:      250,
		}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		var decoded v1.ExecuteQueryRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.GetObjectType(), decoded.GetObjectType())
		assert.Equal(t, original.GetProjectId(), decoded.GetProjectId())
		assert.Equal(t, original.GetLimit(), decoded.GetLimit())
	})

	t.Run("ChatRequest", func(t *testing.T) {
		original := &v1.ChatRequest{
			Message:   "What is the status of project alpha?",
			ProjectId: "proj-alpha",
			AgentId:   "agent-default",
		}
		data, err := proto.Marshal(original)
		require.NoError(t, err)

		var decoded v1.ChatRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.GetMessage(), decoded.GetMessage())
		assert.Equal(t, original.GetProjectId(), decoded.GetProjectId())
		assert.Equal(t, original.GetAgentId(), decoded.GetAgentId())
	})
}

// Verifies that proto3 messages with missing fields (zero values) still
// marshal/unmarshal correctly — proto3 has no "required" concept, so
// missing fields default to zero values.
func TestProtobufMissingRequiredFields(t *testing.T) {
	t.Run("ExecuteQueryRequest with empty fields", func(t *testing.T) {
		req := &v1.ExecuteQueryRequest{
			// ObjectType left empty
			// ProjectId left empty
			// Limit left as zero
		}

		data, err := proto.Marshal(req)
		require.NoError(t, err)

		var decoded v1.ExecuteQueryRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "", decoded.GetObjectType(), "unset string field should default to empty")
		assert.Equal(t, int32(0), decoded.GetLimit(), "unset int32 field should default to 0")
	})

	t.Run("CreateProjectRequest without name", func(t *testing.T) {
		req := &v1.CreateProjectRequest{
			Id: "proj-minimal",
			// Name intentionally left empty
		}
		data, err := proto.Marshal(req)
		require.NoError(t, err)

		var decoded v1.CreateProjectRequest
		err = proto.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, "proj-minimal", decoded.GetId())
		assert.Equal(t, "", decoded.GetName(), "unset string should be empty string")
	})
}
