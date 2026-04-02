---
title: IoT Wireless
description: AWS IoT Wireless emulation in CloudMock
---

## Overview

CloudMock emulates AWS IoT Wireless (LoRaWAN and Sidewalk), supporting wireless devices, gateways, device profiles, service profiles, destinations, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateWirelessDevice | Supported | LoRaWAN or Sidewalk device types |
| GetWirelessDevice | Supported | Returns device details |
| ListWirelessDevices | Supported | Lists all wireless devices |
| UpdateWirelessDevice | Supported | Updates name, destination, description |
| DeleteWirelessDevice | Supported | Deletes a device |
| CreateWirelessGateway | Supported | Creates a LoRaWAN gateway |
| GetWirelessGateway | Supported | Returns gateway details |
| ListWirelessGateways | Supported | Lists all gateways |
| UpdateWirelessGateway | Supported | Updates name and description |
| DeleteWirelessGateway | Supported | Deletes a gateway |
| CreateDeviceProfile | Supported | Validates LoRaWAN RfRegion |
| GetDeviceProfile | Supported | Returns device profile |
| ListDeviceProfiles | Supported | Lists device profiles |
| DeleteDeviceProfile | Supported | Deletes a device profile |
| CreateServiceProfile | Supported | Creates a service profile |
| GetServiceProfile | Supported | Returns service profile |
| ListServiceProfiles | Supported | Lists service profiles |
| DeleteServiceProfile | Supported | Deletes a service profile |
| CreateDestination | Supported | Creates routing destination |
| GetDestination | Supported | Returns destination details |
| ListDestinations | Supported | Lists destinations |
| UpdateDestination | Supported | Updates destination expression |
| DeleteDestination | Supported | Deletes a destination |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

```python
import boto3
client = boto3.client('iotwireless',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

device = client.create_wireless_device(
    Type='LoRaWAN',
    Name='sensor-001',
    DestinationName='my-destination',
)
print(device['Id'])
```

## Configuration

```yaml
services:
  iotwireless:
    enabled: true
```

## Known Differences from AWS

- No actual LoRaWAN or Sidewalk network connectivity
- Valid RfRegion values: US915, EU868, AU915, AS923-1/2/3/4, CN470, CN779, EU433, IN865, KR920, RU864
