// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "testing"

var versionKeys = []string{
	"version",
	"sqlite3.sourceid",
	"sqlite3.versionnumber"
};

func TestVersion(t *testing.T) {
	m, e := Version();

	if (e != nil) {
		t.Fatalf("Version() failed: %s", e);
	}

	for _, k := range versionKeys {
		if _, ok := m[k]; !ok {
			t.Errorf("no \"%s\" key", k);
		}
	}
}
