// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import "testing"
import "os"
import "db"
import "fmt"

const (
	impossibleName = "randomassdatabase.db";
	testName = "testing.db";
)

// Version()

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

// Open()

func openNonexisting(t *testing.T) {
	c, e := Open(impossibleName);
	if (e == nil) {
		fmt.Println(e);
		t.Error("Opened non-existing database");
		c.Close();
	}
}
func openCreate(t *testing.T) {
	c, e := Open(testName+"?"+FlagsURL(OpenCreate));
	if (e != nil) {
		fmt.Println(e);
		t.Error("Failed to create database");
	}
	else {
		c.Close();
	}
}
func openExisting(t *testing.T) {
	c, e := Open(testName+"?"+FlagsURL(OpenReadOnly));
	if (e != nil) {
		fmt.Println(e);
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

// ExecuteDirectly(): tests Prepare() and Execute() in turn
// sets up the database for further tests

type insertTest struct {
	login string;
	password string;
}

var insertTests = []insertTest{
	insertTest{"phf", "somepassword"},
	insertTest{"adt", "somepassword"},
	insertTest{"xyz", "asdfa"},
	insertTest{"abc", "sdfdsdfasdfsdafasdfasdafsdfasd"},
};

func TestCreate(t *testing.T) {
	c, e := Open(testName+"?"+FlagsURL(OpenReadWrite));
	if (e != nil) {
		t.Fatal("Failed to open existing database");
	}

	_, e = db.ExecuteDirectly(c,
		"CREATE TABLE Users("
			"login VARCHAR NOT NULL UNIQUE,"
			"password VARCHAR NOT NULL,"
			"active BOOLEAN NOT NULL DEFAULT 0,"
			"last TIMESTAMP,"
			"PRIMARY KEY (login)"
		")"
	);
	if (e != nil) {
		t.Fatal("Failed to create table");
	}

	for _, k := range insertTests {
		_, e = db.ExecuteDirectly(c,
			"INSERT INTO Users (login, password)"
			"VALUES (?, ?);", k.login, k.password
		);
		if (e != nil) {
			t.Fatal("Failed to insert");
		}
	}

	c.Close();
}

// clean up: remove the test database

func TestDummy(t *testing.T) {
	os.Remove(testName);
}
