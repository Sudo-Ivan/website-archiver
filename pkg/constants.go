// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package pkg contains shared packages and utilities for the website-archiver application.
package pkg

import "time"

const (
	// MaxDepth is the maximum depth for recursive downloads
	MaxDepth = 5

	// HTTPTimeout is the default timeout for HTTP requests
	HTTPTimeout = 30 * time.Second

	// DirPerms is the default directory permissions in octal
	DirPerms = 0750

	// FilePerms is the default file permissions in octal
	FilePerms = 0640

	// File and path constants
	// IllustrationPNG is the name of the illustration file in PNG format
	IllustrationPNG = "illustration.png"
	// DefaultPNG is the name of the default image file
	DefaultPNG = "default.png"
	// IndexHTML is the name of the index HTML file
	IndexHTML = "index.html"
	// ConvertCmd is the name of the image conversion command
	ConvertCmd = "convert"
	// ResizeFlag is the flag used for image resizing
	ResizeFlag = "-resize"
	// ResizeSize is the target size for image resizing
	ResizeSize = "48x48"
	// EmptyString represents an empty string
	EmptyString = ""

	// Log field names
	// LogError is the field name for error logging
	LogError = "error"
	// LogURL is the field name for URL logging
	LogURL = "url"
	// LogTimestamp is the field name for timestamp logging
	LogTimestamp = "timestamp"

	// Array indices
	// FirstIndex represents the first index in an array (0)
	FirstIndex = 0
	// SecondIndex represents the second index in an array (1)
	SecondIndex = 1
	// ThirdIndex represents the third index in an array (2)
	ThirdIndex = 2
	// FourthIndex represents the fourth index in an array (3)
	FourthIndex = 3
	// FifthIndex represents the fifth index in an array (4)
	FifthIndex = 4
	// SixthIndex represents the sixth index in an array (5)
	SixthIndex = 5

	// Wayback Machine URL format
	// WaybackURLFormat is the format string for Wayback Machine URLs
	WaybackURLFormat = "https://web.archive.org/web/%s/%s"

	// CDX API response indices
	// CDXTimestampIndex is the index of the timestamp field in CDX response
	CDXTimestampIndex = 0
	// CDXOriginalIndex is the index of the original URL field in CDX response
	CDXOriginalIndex = 1
	// CDXMimetypeIndex is the index of the mimetype field in CDX response
	CDXMimetypeIndex = 2
	// CDXStatusIndex is the index of the status field in CDX response
	CDXStatusIndex = 3
	// CDXDigestIndex is the index of the digest field in CDX response
	CDXDigestIndex = 4
	// CDXLengthIndex is the index of the length field in CDX response
	CDXLengthIndex = 5

	// Minimum required fields in CDX response
	// MinCDXFields is the minimum number of fields required in a CDX response
	MinCDXFields = 6

	// Minimum required rows in CDX response (header + data)
	// MinCDXRows is the minimum number of rows required in a CDX response
	MinCDXRows = 2

	// Exit codes
	// ExitSuccess represents a successful program exit
	ExitSuccess = 0
	// ExitFailure represents a failed program exit
	ExitFailure = 1

	// Additional constants for magic numbers
	// ZeroDepth represents a depth of zero
	ZeroDepth = 0
	// OneDepth represents a depth of one
	OneDepth = 1
	// ZeroIndex represents an index of zero
	ZeroIndex = 0
	// OneIndex represents an index of one
	OneIndex = 1
	// ZeroValue represents a value of zero
	ZeroValue = 0
	// OneValue represents a value of one
	OneValue = 1
	// ZeroLength represents a length of zero
	ZeroLength = 0
	// OneLength represents a length of one
	OneLength = 1
	// ZeroCount represents a count of zero
	ZeroCount = 0
	// OneCount represents a count of one
	OneCount = 1
	// ZeroPosition represents a position of zero
	ZeroPosition = 0
	// OnePosition represents a position of one
	OnePosition = 1
	// ZeroOffset represents an offset of zero
	ZeroOffset = 0
	// OneOffset represents an offset of one
	OneOffset = 1
	// ZeroIndexPos represents an index position of zero
	ZeroIndexPos = 0
	// OneIndexPos represents an index position of one
	OneIndexPos = 1
	// ZeroArrayPos represents an array position of zero
	ZeroArrayPos = 0
	// OneArrayPos represents an array position of one
	OneArrayPos = 1
	// ZeroSlicePos represents a slice position of zero
	ZeroSlicePos = 0
	// OneSlicePos represents a slice position of one
	OneSlicePos = 1
	// ZeroStringPos represents a string position of zero
	ZeroStringPos = 0
	// OneStringPos represents a string position of one
	OneStringPos = 1
)
