// Copyright 2009 Peter H. Froehlich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlite3

import (
	"db";
	"os";
)

type ResultSet struct {
	// we implement everything in terms of classic stuff
	classic db.ClassicResultSet;
	// channel to send results through
	results chan db.Result;
	// channel to check for termination
	stops chan bool;
}

func (self *ResultSet) init(crs db.ClassicResultSet) {
	self.classic = crs;
	self.results = make(chan db.Result);
	self.stops = make(chan bool);
}

func (self *ResultSet) kill() {
	self.classic.Close();
	self.classic = nil;
	close(self.results);
	self.results = nil;
	close(self.stops);
	self.stops = nil;
}

// goroutine implementing the iterator
func (self *ResultSet) iterate() {
	for self.classic.More() {
		// block until either send or receive
		select {
		case self.results <- self.classic.Fetch():
//			fmt.Printf("sent to %s", self.results);
		case _ = <-self.stops:
//			fmt.Printf("received from %s", self.stops);
			break;
//		default:
//			fmt.Printf("debugging: no communication");
		}
	}
	self.kill();
}

func (self *ResultSet) Iter() <-chan db.Result {
	go self.iterate();
	return self.results;
}

func (self *ResultSet) Close() os.Error {
	if self.stops != nil {
		self.stops <- true;
	}
	return nil;
}

func (self *ResultSet) Names() []string {
	return nil;
}

func (self *ResultSet) Types() []string {
	return nil;
}
