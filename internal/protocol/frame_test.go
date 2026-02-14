package protocol

import (
	"bytes"
	"io"
	"testing"
)

// TestWriteFrameBasic tests writing a simple frame
func TestWriteFrameBasic(t *testing.T) {
	body := []byte("hello world")
	var buf bytes.Buffer

	err := WriteFrame(&buf, body)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	// Expected: 4-byte length (11) + body
	result := buf.Bytes()
	if len(result) != 4+len(body) {
		t.Errorf("expected length %d, got %d", 4+len(body), len(result))
	}

	// Verify length prefix (big-endian)
	expectedLen := uint32(len(body))
	actualLen := uint32(result[0])<<24 | uint32(result[1])<<16 | uint32(result[2])<<8 | uint32(result[3])
	if actualLen != expectedLen {
		t.Errorf("expected length prefix %d, got %d", expectedLen, actualLen)
	}

	// Verify body
	if !bytes.Equal(result[4:], body) {
		t.Errorf("body mismatch: expected %s, got %s", body, result[4:])
	}
}

// TestWriteFrameEmptyBody tests writing empty frame
func TestWriteFrameEmptyBody(t *testing.T) {
	var buf bytes.Buffer
	err := WriteFrame(&buf, []byte{})
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	result := buf.Bytes()
	if len(result) != 4 {
		t.Errorf("expected 4 bytes for empty frame, got %d", len(result))
	}

	// Verify length is 0
	actualLen := uint32(result[0])<<24 | uint32(result[1])<<16 | uint32(result[2])<<8 | uint32(result[3])
	if actualLen != 0 {
		t.Errorf("expected length 0, got %d", actualLen)
	}
}

// TestWriteFrameLargeBody tests writing a large frame
func TestWriteFrameLargeBody(t *testing.T) {
	body := make([]byte, 10000)
	for i := range body {
		body[i] = byte(i % 256)
	}

	var buf bytes.Buffer
	err := WriteFrame(&buf, body)
	if err != nil {
		t.Fatalf("WriteFrame failed: %v", err)
	}

	result := buf.Bytes()
	if len(result) != 4+len(body) {
		t.Errorf("expected length %d, got %d", 4+len(body), len(result))
	}

	if !bytes.Equal(result[4:], body) {
		t.Error("large body mismatch")
	}
}

// TestReadFrameBasic tests reading a simple frame
func TestReadFrameBasic(t *testing.T) {
	// Create frame data manually
	body := []byte("hello world")
	var buf bytes.Buffer

	// Write length + body
	lengthBytes := make([]byte, 4)
	lengthBytes[0] = 0
	lengthBytes[1] = 0
	lengthBytes[2] = 0
	lengthBytes[3] = byte(len(body))
	buf.Write(lengthBytes)
	buf.Write(body)

	// Read frame
	readBuf := make([]byte, 1024)
	result, err := ReadFrame(&buf, readBuf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if !bytes.Equal(result, body) {
		t.Errorf("expected %s, got %s", body, result)
	}
}

// TestReadFrameEmptyBuffer tests reading when buffer slice is provided
func TestReadFrameEmptyBuffer(t *testing.T) {
	body := []byte("test")
	var buf bytes.Buffer

	// Write frame
	lengthBytes := make([]byte, 4)
	lengthBytes[3] = byte(len(body))
	buf.Write(lengthBytes)
	buf.Write(body)

	// Read with small initial buffer - should resize
	readBuf := make([]byte, 0)
	result, err := ReadFrame(&buf, readBuf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if !bytes.Equal(result, body) {
		t.Errorf("expected %s, got %s", body, result)
	}
}

// TestReadFrameExistingBuffer tests reusing existing buffer
func TestReadFrameExistingBuffer(t *testing.T) {
	body := []byte("hello")
	var buf bytes.Buffer

	// Write frame
	lengthBytes := make([]byte, 4)
	lengthBytes[3] = byte(len(body))
	buf.Write(lengthBytes)
	buf.Write(body)

	// Read with small buffer that fits the frame
	readBuf := make([]byte, 10)
	result, err := ReadFrame(&buf, readBuf)
	if err != nil {
		t.Fatalf("ReadFrame failed: %v", err)
	}

	if len(result) != len(body) {
		t.Errorf("expected length %d, got %d", len(body), len(result))
	}
	if !bytes.Equal(result, body) {
		t.Errorf("expected %s, got %s", body, result)
	}
}

// TestReadFrameIncompleteHeader tests error handling when header is incomplete
func TestReadFrameIncompleteHeader(t *testing.T) {
	buf := bytes.NewReader([]byte{0, 0}) // Only 2 bytes instead of 4

	readBuf := make([]byte, 1024)
	_, err := ReadFrame(buf, readBuf)
	if err == nil {
		t.Error("expected error for incomplete header")
	}
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Errorf("expected ErrUnexpectedEOF or EOF, got %v", err)
	}
}

// TestReadFrameIncompleteBody tests error handling when body is incomplete
func TestReadFrameIncompleteBody(t *testing.T) {
	var buf bytes.Buffer

	// Write length (100) but only provide partial body
	lengthBytes := make([]byte, 4)
	lengthBytes[3] = 100
	buf.Write(lengthBytes)
	buf.Write([]byte("hello")) // Only 5 bytes of 100

	readBuf := make([]byte, 1024)
	_, err := ReadFrame(&buf, readBuf)
	if err == nil {
		t.Error("expected error for incomplete body")
	}
	if err != io.ErrUnexpectedEOF && err != io.EOF {
		t.Errorf("expected ErrUnexpectedEOF or EOF, got %v", err)
	}
}

// TestReadFrameMaxFrameSize tests error handling for oversized frame
func TestReadFrameMaxFrameSize(t *testing.T) {
	var buf bytes.Buffer

	// Write length larger than maxFrameSize (1MB)
	lengthBytes := make([]byte, 4)
	lengthBytes[0] = 0xFF
	lengthBytes[1] = 0xFF
	lengthBytes[2] = 0xFF
	lengthBytes[3] = 0xFF
	buf.Write(lengthBytes)

	readBuf := make([]byte, 1024)
	_, err := ReadFrame(&buf, readBuf)
	if err == nil {
		t.Error("expected error for oversized frame")
	}
	if err != io.ErrShortBuffer {
		t.Errorf("expected ErrShortBuffer, got %v", err)
	}
}

// TestReadWriteRoundTrip tests writing and reading back frames
func TestReadWriteRoundTrip(t *testing.T) {
	testCases := []struct {
		name string
		body []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello")},
		{"medium", make([]byte, 1000)},
		{"special chars", []byte{0, 1, 2, 255, 254, 253}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			// Write frame
			err := WriteFrame(&buf, tc.body)
			if err != nil {
				t.Fatalf("WriteFrame failed: %v", err)
			}

			// Read frame
			readBuf := make([]byte, 10000)
			result, err := ReadFrame(&buf, readBuf)
			if err != nil {
				t.Fatalf("ReadFrame failed: %v", err)
			}

			// Verify
			if !bytes.Equal(result, tc.body) {
				t.Errorf("roundtrip failed: expected %d bytes, got %d bytes", len(tc.body), len(result))
			}
		})
	}
}

// TestMultipleFrames tests writing and reading multiple frames
func TestMultipleFrames(t *testing.T) {
	bodies := [][]byte{
		[]byte("frame1"),
		[]byte("frame2"),
		[]byte("frame3"),
	}

	var buf bytes.Buffer

	// Write multiple frames
	for _, body := range bodies {
		err := WriteFrame(&buf, body)
		if err != nil {
			t.Fatalf("WriteFrame failed: %v", err)
		}
	}

	// Read multiple frames
	for i, expectedBody := range bodies {
		readBuf := make([]byte, 1024)
		result, err := ReadFrame(&buf, readBuf)
		if err != nil {
			t.Fatalf("ReadFrame[%d] failed: %v", i, err)
		}

		if !bytes.Equal(result, expectedBody) {
			t.Errorf("frame[%d] mismatch: expected %s, got %s", i, expectedBody, result)
		}
	}
}
