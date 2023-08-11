// Copyright 2021 Atos
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Created on 23 Mar 2022
// Updated on 23 Mar 2022
//
// @author: ATOS
package logs

import (
	log "github.com/sirupsen/logrus"
)

// init
func init() {
	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true})
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.TraceLevel) // TraceLevel, DebugLevel, WarnLevel
}

///////////////////////////////////////////////////////////////////////////////

/*
 */
func Fatal(e error) {
	log.Fatal(e)
}

/*
 */
func Trace(args ...interface{}) {
	log.Trace(args...)
}

/*
 */
func Debug(args ...interface{}) {
	log.Debug(args...)
}

/*
 */
func Info(args ...interface{}) {
	log.Info(args...)
}

/*
 */
func Warn(args ...interface{}) {
	log.Warn(args...)
}

/*
 */
func Error(args ...interface{}) {
	log.Error(args...)
}

/*
 */
func Panic(m string) {
	log.Panic(m)
}
