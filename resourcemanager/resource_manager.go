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

// Package resourcemanager provides the interface to the gateway resource manager API
package resourcemanager

import (
    b64 "encoding/base64"
    "encoding/json"
    "errors"
    "io/ioutil"
    "os"
    "github.com/PelionIoT/edge-resource-manager/resourcemanagerconfig"
    "github.com/op/go-logging"
)

var log = logging.MustGetLogger("EdgeResourceManager")

type gwResourceManagerRegisterArgs struct {
    Name string `json:"name"`
}

type addResouceArgs struct {
    Lwm2mObjects []lwm2mObject `json:"objects"`
}

type lwm2mObject struct {
    ObjectID        int              `json:"objectId"`
    ObjectInstances []objectInstance `json:"objectInstances"`
}

type objectInstance struct {
    ObjectInstanceID int        `json:"objectInstanceId"`
    Resources        []resource `json:"resources"`
}

type resource struct {
    ResourceID int    `json:"resourceId"`
    Operations int    `json:"operations"`
    Type       string `json:"type"`
    Value      string `json:"value"`
}

var resourceManagementClient *Client = nil            // Edge Resource Manager Client
var gcdConfig *resourcemanagerconfig.ResourceManagerConfig // Edge Capability Discovery config

// Run provides the main execution of the program. It adds resources th the gateway as detailed in the config file and handles updates.
func Run(config *resourcemanagerconfig.ResourceManagerConfig) {
    log.Info("resourceManager.Run")

    if config == nil {
        log.Warning("ResourceManager: No config Provided")
        return
    }
    if config.EdgeCoreSocketPath == "" {
        log.Warning("ResourceManager: edge_core_socketpath not provided in config file. Using default socketpath '/tmp/edge.sock'")
        config.EdgeCoreSocketPath = "/tmp/edge.sock"
    }
    if config.EdgeResources == nil {
        log.Errorf("ResourceManager: No edge resources provided")
        return
    }
    if config.ConfigObjectID <= 0 {
        log.Errorf("ResourceManager: lwm2m objectid not provided")
        return
    }
    var err error
    gcdConfig = config
    log.Infof("ResourceManager: connecting to edge-core")

    resourceManagementClient, err = connect(config.EdgeCoreSocketPath) //connect to edge-core
    if err == nil {
        defer resourceManagementClient.Close()
    } else {
        log.Errorf("ResourceManager: could not connect to edge-core")
        return
    }
    log.Infof("ResourceManager: successfully connected to edge-core")
    objectinstanceid := 0
    for _, edgeResource := range config.EdgeResources {
        err = addResource(config.ConfigObjectID, objectinstanceid, 1, 1, "string", edgeResource.Name)
        if err == nil {
            writeResource(config.ConfigObjectID, objectinstanceid, 1, 3, "string", b64.StdEncoding.EncodeToString([]byte(edgeResource.Name)))
        }
        enable := "0"
        if edgeResource.Enable == true {
            enable = "1"
        }
        err = addResource(config.ConfigObjectID, objectinstanceid, 2, 1, "string", string(enable))
        if err == nil {
            writeResource(config.ConfigObjectID, objectinstanceid, 2, 3, "string", b64.StdEncoding.EncodeToString([]byte(enable)))
        }
        err = addResource(config.ConfigObjectID, objectinstanceid, 3, 3, "string", "Config")
        if err == nil {
            b64Config, err := readConfigFile(edgeResource.ConfigFilePath)
            if err == nil {
                // Known issue - to be fixed in next release. An object of size more than 4096 is being
                // truncated by the gorilla/websocket library and thus edge-core closes the connection on
                // receiving the invalid data
                // Hack: To avoid protocol error, shrink the config resource value to a smaller size
                if len(*b64Config) > 3800 {
                    *b64Config = (*b64Config)[:3800]
                }
                writeResource(config.ConfigObjectID, objectinstanceid, 3, 3, "string", *b64Config)
            } else {
                log.Errorf("ResourceManager: could not read %s config file %s, err: %v", edgeResource.Name, edgeResource.ConfigFilePath, err.Error())
            }
        }
        objectinstanceid = objectinstanceid + 1
    }
    updateResourceLoop()
}

func updateResourceLoop() {
    var c clientRequest
    srvreqchannel, err := resourceManagementClient.RegisterRequestReceiver()
    if err == nil {
        for {
            select {
            case srvreq := <-srvreqchannel:
                json.Unmarshal(srvreq, &c)
                log.Debugf("ResourceManager: Got request from edge-core: %v", c)
                if c.Method == "write" {
                    params := c.Params.(map[string]interface{})
                    uri := params["uri"].(map[string]interface{})
                    objectinstanceid := 0
                    for _, edgeResource := range gcdConfig.EdgeResources {
                        if uri != nil && uri["objectId"].(float64) == float64(gcdConfig.ConfigObjectID) && uri["objectInstanceId"].(float64) == float64(objectinstanceid) && uri["resourceId"].(float64) == 3 {
                            log.Debugf("ResourceManager: Writing %s config file", edgeResource.Name)
                            if edgeResource.ConfigFilePath != "" && writeConfigFile(edgeResource.ConfigFilePath, params["value"].(string)) == nil {
                                okresult := json.RawMessage(`"ok"`)
                                resourceManagementClient.Respond(c.ID, &okresult, nil)
                            }
                        }
                        if uri != nil && uri["objectId"].(float64) == float64(gcdConfig.ConfigObjectID) && uri["objectInstanceId"].(float64) == float64(objectinstanceid) && uri["resourceId"].(float64) == 2 {
                            errorResult := json.RawMessage(`{"code": -32602,"message": "Invalid params."}`)
                            resourceManagementClient.Respond(c.ID, nil, &errorResult)
                        }
                        if uri != nil && uri["objectId"].(float64) == float64(gcdConfig.ConfigObjectID) && uri["objectInstanceId"].(float64) == float64(objectinstanceid) && uri["resourceId"].(float64) == 1 {
                            errorResult := json.RawMessage(`"{"code": -32602,"message": "Invalid params."}"`)
                            resourceManagementClient.Respond(c.ID, nil, &errorResult)
                        }
                        objectinstanceid = objectinstanceid + 1
                    }
                } else {
                    log.Debugf("ResourceManager: Unhandled request: %v", c.Method)
                }
            }
        }
    } else {
        log.Errorf("ResourceManager: Could not register request receiver: %s", err.Error())
    }
}

func readConfigFile(filepath string) (*string, error) {
    config, err := ioutil.ReadFile(filepath)
    if err != nil {
        return nil, err
    }

    b64Config := b64.StdEncoding.EncodeToString(config)
    return &b64Config, nil
}

func writeConfigFile(filepath string, config string) error {
    file, err := os.Create(filepath)
    if err != nil {
        log.Errorf("ResourceManager: could not create config file: %v", err)
        return err
    }

    b64Config, _ := b64.StdEncoding.DecodeString(config)
    file.WriteString(string(b64Config[:]))
    file.Close()
    return nil
}

func connect(edgecorescoketpath string) (*Client, error) {

    client := Dial(edgecorescoketpath, "/1/grm", nil)

    var res string
    err := client.Call("gw_resource_manager_register", gwResourceManagerRegisterArgs{Name: "edge_resource_manager"}, &res)

    log.Debugf("ResourceManager: gw_resource_manager_register response:%v", res)
    if err != nil {
        log.Errorf("ResourceManager: Failed to connect to edge-core %s", err.Error())
        return nil, err
    }
    log.Infof("ResourceManager: Websocket connection established with edge-core")

    return client, nil
}

func addResource(objectid int, objectinstanceid int, resourceid int, operationsAllowed int, resourceType string, resourceValue string) error {

    var res string
    addResource := addResouceArgs{
        Lwm2mObjects: []lwm2mObject{
            lwm2mObject{
                ObjectID: objectid,
                ObjectInstances: []objectInstance{
                    objectInstance{
                        ObjectInstanceID: objectinstanceid,
                        Resources: []resource{
                            resource{
                                ResourceID: resourceid,
                                Operations: operationsAllowed,
                                Type:       resourceType,
                                Value:      resourceValue,
                            },
                        },
                    },
                },
            },
        },
    }

    if resourceManagementClient == nil {
        log.Errorf("ResourceManager: resourceManagementClient is nil")
        return errors.New("resourceManagementClient is nil")
    }

    err := resourceManagementClient.Call("add_resource", addResource, &res)

    log.Debugf("ResourceManager: add_resource response:%v", res)
    if err != nil {
        log.Errorf("ResourceManager: Failed to add resource %s", err.Error())
        return err
    }
    log.Infof("ResourceManager: Lwm2m resource %d/%d/%d added", objectid, objectinstanceid, resourceid)
    return nil

}

func writeResource(objectid int, objectinstanceid int, resourceid int, operationsAllowed int, resourceType string, resourceValue string) error {

    var res string
    writeResource := addResouceArgs{
        Lwm2mObjects: []lwm2mObject{
            lwm2mObject{
                ObjectID: objectid,
                ObjectInstances: []objectInstance{
                    objectInstance{
                        ObjectInstanceID: objectinstanceid,
                        Resources: []resource{
                            resource{
                                ResourceID: resourceid,
                                Operations: operationsAllowed,
                                Type:       resourceType,
                                Value:      resourceValue,
                            },
                        },
                    },
                },
            },
        },
    }

    if resourceManagementClient == nil {
        log.Errorf("ResourceManager: resourceManagementClient is nil")
        return errors.New("resourceManagementClient is nil")
    }

    err := resourceManagementClient.Call("write_resource_value", writeResource, &res)

    log.Debugf("ResourceManager: write response:%v", res)
    if err != nil {
        log.Errorf("ResourceManager: Failed to write resource %s", err.Error())
        return err
    }
    log.Infof("ResourceManager: Lwm2m resource %d/%d/%d added", objectid, objectinstanceid, resourceid)
    return nil

}
