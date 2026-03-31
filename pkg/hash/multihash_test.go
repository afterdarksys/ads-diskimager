package hash

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestMultiHasher(t *testing.T) {
	testData := []byte("Hello, World!")

	// Create multi-hasher
	mh := NewDefaultMultiHasher()

	// Write data
	n, err := mh.Write(testData)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len(testData) {
		t.Fatalf("Expected to write %d bytes, wrote %d", len(testData), n)
	}

	// Get results
	result := mh.Sum()

	// Verify MD5
	expectedMD5 := md5.Sum(testData)
	if result.MD5 != hex.EncodeToString(expectedMD5[:]) {
		t.Errorf("MD5 mismatch: got %s, want %s", result.MD5, hex.EncodeToString(expectedMD5[:]))
	}

	// Verify SHA1
	expectedSHA1 := sha1.Sum(testData)
	if result.SHA1 != hex.EncodeToString(expectedSHA1[:]) {
		t.Errorf("SHA1 mismatch: got %s, want %s", result.SHA1, hex.EncodeToString(expectedSHA1[:]))
	}

	// Verify SHA256
	expectedSHA256 := sha256.Sum256(testData)
	if result.SHA256 != hex.EncodeToString(expectedSHA256[:]) {
		t.Errorf("SHA256 mismatch: got %s, want %s", result.SHA256, hex.EncodeToString(expectedSHA256[:]))
	}
}

func TestMultiHasherSelectiveAlgorithms(t *testing.T) {
	testData := []byte("Test data for selective hashing")

	// Only MD5 and SHA256
	mh := NewMultiHasher("md5", "sha256")

	mh.Write(testData)
	result := mh.Sum()

	// Should have MD5
	if result.MD5 == "" {
		t.Error("Expected MD5 hash, got empty string")
	}

	// Should have SHA256
	if result.SHA256 == "" {
		t.Error("Expected SHA256 hash, got empty string")
	}

	// Should NOT have SHA1
	if result.SHA1 != "" {
		t.Error("Expected empty SHA1 (not enabled), got value")
	}
}

func TestMultiHasherGetHash(t *testing.T) {
	testData := []byte("Data for specific hash retrieval")
	mh := NewMultiHasher("md5", "sha1")

	mh.Write(testData)

	// Get MD5
	md5Hash, err := mh.GetHash("md5")
	if err != nil {
		t.Errorf("Failed to get MD5: %v", err)
	}
	if md5Hash == "" {
		t.Error("Got empty MD5 hash")
	}

	// Get SHA1
	sha1Hash, err := mh.GetHash("sha1")
	if err != nil {
		t.Errorf("Failed to get SHA1: %v", err)
	}
	if sha1Hash == "" {
		t.Error("Got empty SHA1 hash")
	}

	// Try to get SHA256 (not enabled)
	_, err = mh.GetHash("sha256")
	if err == nil {
		t.Error("Expected error when getting disabled algorithm, got nil")
	}
}

func TestHashReader(t *testing.T) {
	testData := "Test data for reader hashing"
	reader := strings.NewReader(testData)

	result, err := HashReader(reader, "md5", "sha256")
	if err != nil {
		t.Fatalf("HashReader failed: %v", err)
	}

	if result.MD5 == "" {
		t.Error("Expected MD5 hash from reader")
	}
	if result.SHA256 == "" {
		t.Error("Expected SHA256 hash from reader")
	}
	if result.SHA1 != "" {
		t.Error("SHA1 should be empty (not requested)")
	}
}

func TestMultiHasherConcurrentWrites(t *testing.T) {
	mh := NewDefaultMultiHasher()

	// Write in chunks
	chunk1 := []byte("First chunk ")
	chunk2 := []byte("Second chunk ")
	chunk3 := []byte("Third chunk")

	mh.Write(chunk1)
	mh.Write(chunk2)
	mh.Write(chunk3)

	result := mh.Sum()

	// Compare with single write
	fullData := append(chunk1, chunk2...)
	fullData = append(fullData, chunk3...)

	expectedMD5 := md5.Sum(fullData)
	if result.MD5 != hex.EncodeToString(expectedMD5[:]) {
		t.Error("Multi-write MD5 doesn't match single-write MD5")
	}
}

func TestMultiHasherAlgorithms(t *testing.T) {
	mh := NewMultiHasher("md5", "sha1")

	algos := mh.Algorithms()
	if len(algos) != 2 {
		t.Errorf("Expected 2 algorithms, got %d", len(algos))
	}

	// Check HasAlgorithm
	if !mh.HasAlgorithm("md5") {
		t.Error("Expected md5 to be present")
	}
	if !mh.HasAlgorithm("sha1") {
		t.Error("Expected sha1 to be present")
	}
	if mh.HasAlgorithm("sha256") {
		t.Error("sha256 should not be present")
	}
}

func BenchmarkMultiHasher(b *testing.B) {
	data := bytes.Repeat([]byte("benchmark data "), 1024) // ~15KB

	b.Run("MultiHash-All", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			mh := NewDefaultMultiHasher()
			mh.Write(data)
			_ = mh.Sum()
		}
	})

	b.Run("SingleHash-MD5", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			h := md5.New()
			h.Write(data)
			_ = h.Sum(nil)
		}
	})

	b.Run("SingleHash-SHA256", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		for i := 0; i < b.N; i++ {
			h := sha256.New()
			h.Write(data)
			_ = h.Sum(nil)
		}
	})
}
