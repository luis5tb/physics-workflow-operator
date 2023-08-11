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
package common

import (
	"encoding/json"

	log "gogs.apps.ocphub.physics-faas.eu/wp5/physics-workflow-operator/common/logs"
)

// path used in logs
const pathLOG string = "OW-PROXY > Common "

/*
StructToString Parses a struct to a string
*/
func StructToString(ct interface{}) (string, error) {
	out, err := json.Marshal(ct)
	if err != nil {
		log.Error(pathLOG+"[StructToString] ERROR ", err)
		return "", err
	}

	return string(out), nil
}

/*
LogStruct Parses a struct to a string and shows the content
*/
func LogStruct(pathlog string, name string, ct interface{}) {
	out, err := json.Marshal(ct)
	if err != nil {
		log.Error(pathLOG+"[LogStruct] ERROR ", err)
	} else {
		log.Trace(pathlog + " [" + name + "] : [" + string(out) + "]")
	}
}
