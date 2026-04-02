---
title: Dart / Flutter
description: Using CloudMock with Dart, Flutter, and the AWS SDK
---

CloudMock does not require a custom Dart SDK. Configure the AWS SDK for Dart or your HTTP client to point at the CloudMock gateway. This works for Flutter mobile apps, Flutter web, and server-side Dart.

> **Note:** CloudMock does not have a dedicated Dart SDK adapter yet. Use the HTTP mode by starting CloudMock with `npx cloudmock` and pointing your AWS SDK at `http://localhost:4566`.

## AWS SDK for Dart (aws_common / smithy)

The official AWS SDK for Dart is still in developer preview. For services that have a published Dart package, configure the endpoint:

```yaml
# pubspec.yaml
dependencies:
  aws_s3_api: ^0.2.0        # if available
  aws_dynamodb_api: ^0.2.0   # if available
  aws_common: ^0.7.0
```

### Endpoint configuration

```dart
import 'package:aws_s3_api/s3-2006-03-01.dart';
import 'package:aws_common/aws_common.dart';

final credentials = AWSCredentialsProvider(AWSCredentials(
  'test',  // accessKeyId
  'test',  // secretAccessKey
));

final s3 = S3(
  region: 'us-east-1',
  credentialsProvider: credentials,
  endpointUrl: Uri.parse('http://localhost:4566'),
);
```

## Using the http package directly

If the official AWS Dart SDK does not cover the service you need, you can call CloudMock's REST API directly using Dart's `http` package or `dio`:

### http package

```dart
import 'package:http/http.dart' as http;
import 'dart:convert';

const cloudmockUrl = 'http://localhost:4566';

// Create an S3 bucket
Future<void> createBucket(String name) async {
    final response = await http.put(
        Uri.parse('$cloudmockUrl/$name'),
        headers: {
            'Authorization': 'AWS4-HMAC-SHA256 ...',
            'x-amz-content-sha256': 'UNSIGNED-PAYLOAD',
        },
    );
    print('Create bucket: ${response.statusCode}');
}

// List S3 buckets
Future<void> listBuckets() async {
    final response = await http.get(
        Uri.parse(cloudmockUrl),
        headers: {
            'Authorization': 'AWS4-HMAC-SHA256 ...',
        },
    );
    print('Buckets: ${response.body}');
}
```

### dio

```dart
import 'package:dio/dio.dart';

final dio = Dio(BaseOptions(
    baseUrl: 'http://localhost:4566',
    headers: {
        'Authorization': 'AWS4-HMAC-SHA256 ...',
    },
));

// DynamoDB - CreateTable
Future<void> createTable() async {
    final response = await dio.post(
        '/',
        data: {
            'TableName': 'Users',
            'KeySchema': [
                {'AttributeName': 'UserId', 'KeyType': 'HASH'}
            ],
            'AttributeDefinitions': [
                {'AttributeName': 'UserId', 'AttributeType': 'S'}
            ],
            'BillingMode': 'PAY_PER_REQUEST',
        },
        options: Options(headers: {
            'X-Amz-Target': 'DynamoDB_20120810.CreateTable',
            'Content-Type': 'application/x-amz-json-1.0',
        }),
    );
    print('Table created: ${response.data}');
}
```

## Testing with dart test

Use the `test` package for Dart tests. Start CloudMock before running the test suite, or use environment variables to configure the endpoint.

```dart
import 'package:test/test.dart';
import 'package:aws_s3_api/s3-2006-03-01.dart';
import 'package:aws_dynamodb_api/dynamodb-2012-08-10.dart';
import 'package:aws_common/aws_common.dart';
import 'package:http/http.dart' as http;

const cloudmockEndpoint = 'http://localhost:4566';
const adminEndpoint = 'http://localhost:4599';

Future<void> resetCloudMock() async {
    await http.post(Uri.parse('$adminEndpoint/api/reset'));
}

void main() {
    late S3 s3;
    late DynamoDB ddb;

    setUp(() async {
        await resetCloudMock();

        final credentials = AWSCredentialsProvider(AWSCredentials('test', 'test'));

        s3 = S3(
            region: 'us-east-1',
            credentialsProvider: credentials,
            endpointUrl: Uri.parse(cloudmockEndpoint),
        );

        ddb = DynamoDB(
            region: 'us-east-1',
            credentialsProvider: credentials,
            endpointUrl: Uri.parse(cloudmockEndpoint),
        );
    });

    test('S3 create and list buckets', () async {
        await s3.createBucket(bucket: 'test');
        final result = await s3.listBuckets();
        final names = result.buckets?.map((b) => b.name).toList() ?? [];
        expect(names, contains('test'));
    });

    test('DynamoDB create table and put item', () async {
        await ddb.createTable(
            tableName: 'users',
            keySchema: [KeySchemaElement(attributeName: 'pk', keyType: KeyType.hash)],
            attributeDefinitions: [
                AttributeDefinition(
                    attributeName: 'pk',
                    attributeType: ScalarAttributeType.s,
                )
            ],
            billingMode: BillingMode.payPerRequest,
        );

        await ddb.putItem(
            tableName: 'users',
            item: {
                'pk': AttributeValue(s: 'user-1'),
                'name': AttributeValue(s: 'Alice'),
            },
        );

        final resp = await ddb.getItem(
            tableName: 'users',
            key: {'pk': AttributeValue(s: 'user-1')},
        );

        expect(resp.item?['name']?.s, equals('Alice'));
    });
}
```

Start CloudMock before running:

```bash
npx cloudmock start &
dart test
```

## Community packages

Several community packages provide higher-level AWS clients for Dart:

- **`aws_s3_api`** -- S3 operations with Dart types
- **`aws_dynamodb_api`** -- DynamoDB operations
- **`aws_cognito_identity_provider`** -- Cognito User Pools

Most accept an `endpointUrl` parameter. Set it to `http://localhost:4566` (or the appropriate host for your environment).

## Flutter networking

### Android emulator

The Android emulator cannot reach `localhost` on the host machine. Use `10.0.2.2` instead:

```dart
const cloudmockUrl = 'http://10.0.2.2:4566';
```

### iOS Simulator

The iOS Simulator shares the host Mac's network stack. `localhost` works directly:

```dart
const cloudmockUrl = 'http://localhost:4566';
```

### Platform-aware endpoint

```dart
import 'dart:io' show Platform;

String get cloudmockUrl {
    if (Platform.isAndroid) {
        return 'http://10.0.2.2:4566';
    }
    return 'http://localhost:4566';
}
```

### Flutter Web

When running `flutter run -d chrome`, the app runs in the browser on the same machine as CloudMock. Use `http://localhost:4566`. CORS headers are returned by CloudMock by default.

### Cleartext traffic (Android)

Android blocks cleartext HTTP by default. Add to `android/app/src/main/AndroidManifest.xml`:

```xml
<application
    android:usesCleartextTraffic="true"
    ... >
```

Or use a more targeted network security config as described in the [Kotlin guide](/docs/language-guides/kotlin/#network-security-config).

### App Transport Security (iOS)

Add to `ios/Runner/Info.plist`:

```xml
<key>NSAppTransportSecurity</key>
<dict>
    <key>NSAllowsLocalNetworking</key>
    <true/>
</dict>
```

## Signing requests

AWS APIs require Signature V4 signed requests. If you are calling CloudMock directly (without an AWS SDK), you have two options:

1. **Set IAM mode to `none`** in CloudMock configuration, which bypasses all authentication:

   ```yaml
   iam:
     mode: none
   ```

   Then you can make unsigned requests. This is the simplest approach for development.

2. **Use an AWS SigV4 signing library** for Dart. The `aws_common` package provides signing utilities.

When using an official AWS SDK package with `endpointUrl`, signing is handled automatically.

## Common issues

### AWS SDK maturity

The official AWS SDK for Dart is in developer preview. Not all services have published packages. For services without a Dart SDK, use direct HTTP calls with the `X-Amz-Target` header for JSON-protocol services or REST-style URLs for REST-XML services.

### Emulator IP addresses

Use `10.0.2.2` for Android Emulator and `localhost` for iOS Simulator. Physical devices require the host machine's IP on the local network.
