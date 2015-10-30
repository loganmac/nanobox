// Copyright (c) 2015 Pagoda Box Inc
//
// This Source Code Form is subject to the terms of the Mozilla Public License, v.
// 2.0. If a copy of the MPL was not distributed with this file, You can obtain one
// at http://mozilla.org/MPL/2.0/.
//

//
package vagrant

import (
	"fmt"
	"os"

	"github.com/jcelliott/lumber"

	"github.com/nanobox-io/nanobox/config"
)

//
var (
	Console *lumber.ConsoleLogger
	Log     *lumber.FileLogger
)

// NewLogger
func NewLogger(path string) {

	// create a console logger
	Console = lumber.NewConsoleLogger(lumber.INFO)

	// create a file logger
	if Log, err = lumber.NewAppendLogger(path); err != nil {
		config.Error("Failed to create a Vagrant logger", err.Error())
	}
}

// Info
func Info(msg string, debug bool) {
	Log.Info(msg)
}

// Debug
func Debug(msg string, debug bool) {
	if debug {
		fmt.Printf(msg)
	}
}

// Fatal
func Fatal(msg, err string) {
	fmt.Printf("A Vagrant error occurred (See %s for details). Exiting...", config.AppDir+"/vagrant.log")
	Log.Fatal(fmt.Sprintf("%s - %s", msg, err))
	Log.Close()
	os.Exit(1)
}
