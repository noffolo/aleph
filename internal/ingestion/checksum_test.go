package ingestion

import (
	"testing"
)

func TestComputeChecksum(t *testing.T) {
	data := []byte("test data for checksum")
	checksum := computeChecksum(data)

	if checksum == "" {
		t.Fatal("checksum should not be empty")
	}

	if len(checksum) != 64 {
		t.Fatalf("checksum should be 64 characters (SHA-256 hex), got %d", len(checksum))
	}
}

func TestComputeChecksumDeterministic(t *testing.T) {
	data := []byte("consistent test data")
	checksum1 := computeChecksum(data)
	checksum2 := computeChecksum(data)

	if checksum1 != checksum2 {
		t.Fatal("checksum should be deterministic for same input")
	}
}

func TestComputeChecksumDifferentData(t *testing.T) {
	data1 := []byte("first test data")
	data2 := []byte("second test data")

	checksum1 := computeChecksum(data1)
	checksum2 := computeChecksum(data2)

	if checksum1 == checksum2 {
		t.Fatal("different data should produce different checksums")
	}
}

func TestVerifyChecksum(t *testing.T) {
	data := []byte("test data for verification")
	checksum := computeChecksum(data)

	if !VerifyChecksum(data, checksum) {
		t.Fatal("verification should succeed for matching checksum")
	}
}

func TestVerifyChecksumMismatch(t *testing.T) {
	data := []byte("test data")
	wrongChecksum := "0000000000000000000000000000000000000000000000000000000000000000"

	if VerifyChecksum(data, wrongChecksum) {
		t.Fatal("verification should fail for mismatched checksum")
	}
}

func TestVerifyChecksumEmptyData(t *testing.T) {
	data := []byte{}
	checksum := computeChecksum(data)

	if !VerifyChecksum(data, checksum) {
		t.Fatal("verification should succeed for empty data with valid checksum")
	}
}
