# Edge Resource Manager

The Edge Resource Manager adds LwM2M resources to an Edge device at runtime.

It achieves this by communicating to edge-core using the gateway resource manager REST API.

The API is detailed in [grm-json-rpc](https://developer.izumanetworks.com/docs/device-management-edge/latest/managing/grm-json-rpc.html)

## Configuration

The edge-resource manager takes a configuration file as it's input that describes the resources that are to be added to edge-core.

The file [`izuma-base-config.yaml`](izuma-base-config.yaml) provides an example of this configuration file. It is expected that each use case of edge-resource-manager provides their own implementation of the configuration file.

The configuration file contains information about how to connect to edge-core and the LwM2M resources than should be added.

The **name** and **enable** fields of each LwM2M resource are read only resources. They cannot be changed at runtime.

## Execution

The edge-resource-manager should be started as a systemd service on the gateway.

A typical service file is shown below:

```
[Unit]
Description=edge-resource-manager: Adds gateway capability resources.
Requires=edge-proxy.service
After=edge-proxy.service

[Service]
Restart=always
RestartSec=5s
ExecStart=/usr/bin/edge-resource-manager -config /etc/edge/izuma-base-config.yaml
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```
