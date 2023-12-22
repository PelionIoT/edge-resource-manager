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

// Package resourcemanagerconfig provides the abstraction for reading the configuration file
package resourcemanagerconfig

import (
    yaml "gopkg.in/yaml.v2"
    "io/ioutil"
    "github.com/op/go-logging"
)

var log = logging.MustGetLogger("EdgeResourceManager")

/**
 * These are used by the config file, and in some cases by
 * the API
 */

// YAMLConfig provides the overlay definition for the config file structure
type YAMLConfig struct {
    EdgeCapabilities  *ResourceManagerConfig  `yaml:"edge_capabilities"`
    ConfigEnd         bool        `yaml:"config_end"`
}

// ResourceManagerConfig provides the overlay for the edge capabilities
type ResourceManagerConfig struct {
    EdgeCoreSocketPath string         `yaml:"edge_core_socketpath"`
    ConfigObjectID     int            `yaml:"lwm2m_objectid"`
    EdgeResources      []EdgeResource `yaml:"edge_resources"`
}

// EdgeResource provides the overlay for a single resource.
type EdgeResource struct {
    Name           string `yaml:"name"`
    Enable         bool   `yaml:"enable"`
    ConfigFilePath string `yaml:"config_filepath"`
}


// LoadFromFile load the config file
func (ysc *YAMLConfig) LoadFromFile(file string) error {
    log.Info("Loading config file:", file)
    rawConfig, err := ioutil.ReadFile(file)

    if err != nil {
        log.Error("Failed to load config file:", err)
        return err
    }

    err = yaml.Unmarshal(rawConfig, ysc)

    if err != nil {
        log.Error("Failed to parse config:", err)
        return err
    }

    if ysc.ConfigEnd != true {
        log.Error("Did not see a \"config_end: true\" statement at end. Possible bad parse??")
    }


    return nil
}
