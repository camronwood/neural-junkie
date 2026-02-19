package repo

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
)

// CompressContent compresses a string using gzip and encodes it as base64
func CompressContent(content string) (string, int64, error) {
	var buf bytes.Buffer

	// Create gzip writer
	gzWriter := gzip.NewWriter(&buf)

	// Write content
	_, err := gzWriter.Write([]byte(content))
	if err != nil {
		return "", 0, fmt.Errorf("failed to write to gzip: %w", err)
	}

	// Close gzip writer to flush
	if err := gzWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Encode to base64
	compressed := buf.Bytes()
	encoded := base64.StdEncoding.EncodeToString(compressed)

	return encoded, int64(len(compressed)), nil
}

// DecompressContent decodes base64 and decompresses gzip content
func DecompressContent(compressed string) (string, error) {
	// Decode from base64
	decoded, err := base64.StdEncoding.DecodeString(compressed)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Create gzip reader
	gzReader, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Read decompressed content
	decompressed, err := io.ReadAll(gzReader)
	if err != nil {
		return "", fmt.Errorf("failed to read decompressed content: %w", err)
	}

	return string(decompressed), nil
}

// CompressionStats tracks compression statistics
type CompressionStats struct {
	OriginalSize   int64
	CompressedSize int64
	FileCount      int
}

// CompressionRatio returns the compression ratio as a percentage
func (cs *CompressionStats) CompressionRatio() float64 {
	if cs.OriginalSize == 0 {
		return 0
	}
	return float64(cs.CompressedSize) / float64(cs.OriginalSize) * 100
}

// SpaceSaved returns the amount of space saved in bytes
func (cs *CompressionStats) SpaceSaved() int64 {
	return cs.OriginalSize - cs.CompressedSize
}

// FormatSize formats a byte size in a human-readable format
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
