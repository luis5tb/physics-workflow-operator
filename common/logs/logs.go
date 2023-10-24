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
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

var cluster_key string

// init
func init() {
	log_level := os.Getenv("LOG_LEVEL")

	log.SetFormatter(&log.TextFormatter{ForceColors: true, FullTimestamp: true}) //log.SetFormatter(&log.JSONFormatter{})

	if len(log_level) == 0 {
		log.SetLevel(log.InfoLevel)
	} else {
		if strings.ToLower(log_level) == "trace" {
			log.SetLevel(log.TraceLevel)
		} else if strings.ToLower(log_level) == "debug" {
			log.SetLevel(log.DebugLevel)
		} else {
			log.SetLevel(log.InfoLevel)
		}
	}

	cluster_key = os.Getenv("PHYSICS_CLUSTER_KEY")
	if len(log_level) > 0 {
		cluster_key = "[" + cluster_key + "] "
	}
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
	args = append([]interface{}{cluster_key}, args...)
	log.Trace(args...)
}

/*
 */
func Debug(args ...interface{}) {
	args = append([]interface{}{cluster_key}, args...)
	log.Debug(args...)
}

/*
 */
func Info(args ...interface{}) {
	args = append([]interface{}{cluster_key}, args...)
	log.Info(args...)
}

/*
 */
func Warn(args ...interface{}) {
	args = append([]interface{}{cluster_key}, args...)
	log.Warn(args...)
}

/*
 */
func Error(args ...interface{}) {
	args = append([]interface{}{cluster_key}, args...)
	log.Error(args...)
}

/*
 */
func Panic(m string) {
	log.Panic(m)
}
