/*
Copyright (c) 2023, Izuma Networks

SPDX-License-Identifier: Apache-2.0
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
    "flag"
    "os"
    "github.com/op/go-logging"
    "github.com/PelionIoT/edge-resource-manager/resourcemanagerconfig"
    "github.com/PelionIoT/edge-resource-manager/resourcemanager"
)

var log = logging.MustGetLogger("EdgeResourceManager")
func init() {
    backend := logging.NewLogBackend(os.Stderr, "", 0)
    format := logging.MustStringFormatter(
        `%{color}%{time:15:04:05.000} ▶ %{level:.4s} %{callpath}%{color:reset} %{message}`,
    )
    backendFormatter := logging.NewBackendFormatter(backend, format)
    logging.SetBackend(backendFormatter)
}


func main() {

    log.Info("edge-resource-manager starting.")

    configFlag := flag.String("config", "./izuma-base-config.yaml", "Config path")
    flag.Parse()

    if configFlag != nil {
        log.Infof("config file: %s\n", *configFlag)
    }

    // Initialization starts off with reading in the entire config file,
    // which also creates and populated the config macro variable dictionary,
    // which is used in different parts of the configs

    config := new(resourcemanagerconfig.YAMLConfig)
    err := config.LoadFromFile(*configFlag)

    if err != nil {
        log.Errorf("Critical error. Config file parse failed --> %s\n", err.Error())
        os.Exit(1)
    }

    resourcemanager.Run(config.EdgeCapabilities)

}

