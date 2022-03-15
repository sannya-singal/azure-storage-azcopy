// Copyright © 2017 Microsoft <wastore@microsoft.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Version struct {
	segments []int64
	preview  bool
	original string
}

// To keep the code simple, we assume we only use a simple subset of semantic versions.
// Namely, the version is either a normal stable version, or a pre-release version with '-preview' attached.
// Examples: 10.1.0, 11.2.0-preview
func NewVersion(raw string) (*Version, error) {
	const standardError = "invalid version string"

	rawSegments := strings.Split(raw, ".")
	if len(rawSegments) != 3 {
		return nil, errors.New(standardError)
	}

	v := &Version{segments: make([]int64, 3), original: raw}
	for i, str := range rawSegments {
		if strings.Contains(str, "-") {
			if i != 2 {
				return nil, errors.New(standardError)
			}
			v.preview = true
			str = strings.Split(str, "-")[0]
		}

		val, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return nil, errors.New("cannot version string")
		}
		v.segments[i] = val
	}

	return v, nil
}

// compare this version (v) to another version (v2)
// return -1 if v is smaller/older than v2
// return 0 if v is equal to v2
// return 1 if v is bigger/newer than v2
func (v Version) compare(v2 Version) int {
	// short-circuit if the two version have the exact same raw string, no need to compare
	if v.original == v2.original {
		return 0
	}

	// compare the major/minor/patch version
	// if v has a bigger number, it is newer
	for i, num := range v.segments {
		if num > v2.segments[i] {
			return 1
		} else if num < v2.segments[i] {
			return -1
		}
	}

	// if both or neither versions are previews, then they are equal
	// usually this shouldn't happen since we already checked whether the two versions have equal raw string
	// however, it is entirely possible that we have new kinds of pre-release versions that this code is not parsing correctly
	// in this case we consider both pre-release version equal anyways
	if (v.preview && v2.preview) || (!v.preview && !v2.preview) {
		return 0
	} else if v.preview && !v2.preview {
		return -1
	}

	return 1
}

// OlderThan detect if version v is older than v2
func (v Version) OlderThan(v2 Version) bool {
	return v.compare(v2) == -1
}

// NewerThan detect if version v is newer than v2
func (v Version) NewerThan(v2 Version) bool {
	return v.compare(v2) == 1
}

// CacheNewerVersion caches the version v2 to filePath if v2 is newer than v1
func (v Version) CacheNewerVersion(v2 Version, filePath string) {
	if v.OlderThan(v2) {
		expiry := time.Now().Add(24 * time.Hour).Format(versionFileTimeFormat)
		if err := os.WriteFile(filePath, []byte(v2.original+","+expiry), 0666); err != nil {
			fmt.Println(err)
		}
	}
}

// ValidateCachedVersion checks if the given filepath contains cached version, expiry or not.
// If yes, then it reads the cache, checks if the cache is still fresh and finally creates Version object from it and returns it.
func ValidateCachedVersion(filePath string) (*Version, error) {
	// Check the locally cached file to get the version.
	data, err := os.ReadFile(filePath)
	if err == nil {
		// If the data is fresh, don't make the call and return right away
		versionAndExpiry := strings.Split(fmt.Sprintf("%s", data), ",")
		if len(versionAndExpiry) == 2 {
			version, err := NewVersion(versionAndExpiry[0])
			if err == nil {
				expiry, err := time.Parse(versionFileTimeFormat, versionAndExpiry[1])
				if err == nil && expiry.After(time.Now()) {
					return version, nil
				}
			}
		}
	}
	return nil, errors.New("failed to fetch or validate the cached version")
}
