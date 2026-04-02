---
title: IoT Wireless
description: AWS IoT Wireless emulation in CloudMock
---

## Overview

CloudMock emulates AWS IoT Wireless, supporting wireless devices, gateways, device profiles, service profiles, destinations, and tagging.

## Supported Operations

| Operation | Status | Notes |
|-----------|--------|-------|
| CreateWirelessDevice | Supported | Creates a wireless device |
| GetWirelessDevice | Supported | Returns device details |
| ListWirelessDevices | Supported | Lists all wireless devices |
| DeleteWirelessDevice | Supported | Deletes a wireless device |
| UpdateWirelessDevice | Supported | Updates a wireless device |
| CreateWirelessGateway | Supported | Creates a wireless gateway |
| GetWirelessGateway | Supported | Returns gateway details |
| ListWirelessGateways | Supported | Lists all wireless gateways |
| DeleteWirelessGateway | Supported | Deletes a wireless gateway |
| UpdateWirelessGateway | Supported | Updates a wireless gateway |
| CreateDeviceProfile | Supported | Creates a device profile |
| GetDeviceProfile | Supported | Returns device profile details |
| ListDeviceProfiles | Supported | Lists device profiles |
| DeleteDeviceProfile | Supported | Deletes a device profile |
| CreateServiceProfile | Supported | Creates a service profile |
| GetServiceProfile | Supported | Returns service profile details |
| ListServiceProfiles | Supported | Lists service profiles |
| DeleteServiceProfile | Supported | Deletes a service profile |
| CreateDestination | Supported | Creates a destination |
| GetDestination | Supported | Returns destination details |
| ListDestinations | Supported | Lists all destinations |
| UpdateDestination | Supported | Updates a destination |
| DeleteDestination | Supported | Deletes a destination |
| TagResource | Supported | Adds tags to a resource |
| UntagResource | Supported | Removes tags from a resource |
| ListTagsForResource | Supported | Lists tags for a resource |

## Quick Start

### Node.js

```typescript
import { IoTWirelessClient, CreateWirelessDeviceCommand } from '@aws-sdk/client-iot-wireless';

const client = new IoTWirelessClient({
  endpoint: 'http://localhost:4566',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

const { Id, Arn } = await client.send(new CreateWirelessDeviceCommand({
  Type: 'LoRaWAN',
  DestinationName: 'my-destination',
}));
console.log(Id);
```

### Python

```python
import boto3

client = boto3.client('iotwireless',
    endpoint_url='http://localhost:4566',
    region_name='us-east-1',
    aws_access_key_id='test',
    aws_secret_access_key='test')

response = client.create_wireless_device(
    Type='LoRaWAN',
    DestinationName='my-destination')
print(response['Id'])
```

## Configuration

```yaml
# cloudmock.yml
services:
  iotwireless:
    enabled: true
```

## Known Differences from AWS

- No actual LoRaWAN or Sidewalk connectivity
- Devices and gateways are metadata-only
- Message routing through destinations is not performed
