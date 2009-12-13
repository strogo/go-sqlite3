// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// SQLite database driver for Go.
//
// Please see http://www.sqlite.org/c3ref/intro.html for all
// the missing details. Sorry, our documentation is focused
// on driver details, not on SQLite in general.
//
// Restrictions on Types:
//
// For now, we treat *all* values as strings. This is less of
// an issue for SQLite since it's typed dynamically anyway.
// Accepting/returning appropriate Go types is a high priority
// goal though.
//
// Binding Query Parameters:
//
// SQL queries can contain "?" parameter slots that are bound
// to values in Execute(). SQLite has more variations that we
// don't support for now: they would make the API more complex
// for very little gain. Parameter slots are matched to values
// in order of appearance.
//
// Concurrency:
//
// We still need to address concurrency issues in detail, for
// now we simply force SQLite into "serialized" threading mode
// (see http://www.sqlite.org/threadsafe.html for details).
//
//
// Low-Level API:
//
// The file low.go contains the low-level API for SQLite. The
// API is not exposed, and you should only have to worry about
// it if you're hunting for bugs in the database driver. Note
// that the technical reason for the low-level API is that cgo
// can't process multiple files at once (a factoid that really
// doesn't have any bearing whatsoever on applications).
package sqlite3
