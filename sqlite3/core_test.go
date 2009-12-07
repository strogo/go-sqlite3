// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "testing"
import "os"

const (
	impossibleName = "randomassdatabase.db";
	testName = "testing.db";
)

type versionTest struct {
	key		string;
	required	bool;
}

var versionTests = []versionTest{
	versionTest{"version", true},
	versionTest{"sqlite3.sourceid", false},
	versionTest{"sqlite3.versionnumber", true},
}

func TestVersion(t *testing.T) {
	m, e := Version();

	if e != nil {
		t.Fatalf("Version() failed: %s", e)
	}

	for _, k := range versionTests {
		if _, ok := m[k.key]; !ok {
			if k.required {
				t.Errorf("no \"%s\" key", k)
			} else {
				t.Logf("no \"%s\" key", k)
			}
		}
	}
}

func openNonexisting(t *testing.T) {
	c, e := Open(ConnectionInfo{"name": impossibleName});
	if (e == nil) {
		t.Error("Opened non-existing database");
		c.Close();
	}
}
func openCreate(t *testing.T) {
	c, e := Open(ConnectionInfo{"name": testName, "sqlite3.flags": OpenCreate});
	if (e != nil) {
		t.Error("Failed to create database");
	}
	else {
		c.Close();
	}
}
func openExisting(t *testing.T) {
	c, e := Open(ConnectionInfo{"name": testName, "sqlite3.flags": OpenReadOnly});
	if (e != nil) {
		t.Error("Failed to open existing database");
	}
	else {
		c.Close();
	}
}

func TestOpen(t *testing.T) {
	openNonexisting(t);
	openCreate(t);
	openExisting(t);
}

// just to clean up and remove the database
func TestDummy(t *testing.T) {
	os.Remove(testName);
}
