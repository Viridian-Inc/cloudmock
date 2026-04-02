---
title: C / C++
description: Using CloudMock with the AWS SDK for C++
---

# C / C++

## CloudMock C SDK

The CloudMock C SDK provides lifecycle management:

```c
#include <cloudmock.h>

cloudmock_t *cm = cloudmock_start(NULL);
printf("Endpoint: %s\n", cloudmock_endpoint(cm));

// Use AWS SDK for C++ with this endpoint

cloudmock_stop(cm);
```

Build: `make` in the `sdk/cpp/` directory produces `libcloudmock.a`.

## AWS SDK for C++ Example

```cpp
#include <cloudmock.h>
#include <aws/core/Aws.h>
#include <aws/s3/S3Client.h>
#include <aws/s3/model/CreateBucketRequest.h>

int main() {
    auto *cm = cloudmock_start(nullptr);

    Aws::SDKOptions options;
    Aws::InitAPI(options);
    {
        Aws::Client::ClientConfiguration config;
        config.endpointOverride = cloudmock_endpoint(cm);
        config.scheme = Aws::Http::Scheme::HTTP;
        config.region = "us-east-1";

        auto creds = Aws::MakeShared<Aws::Auth::SimpleAWSCredentialsProvider>(
            "alloc", "test", "test");

        Aws::S3::S3Client s3(creds, nullptr, config);

        Aws::S3::Model::CreateBucketRequest req;
        req.SetBucket("my-bucket");
        s3.CreateBucket(req);
    }
    Aws::ShutdownAPI(options);

    cloudmock_stop(cm);
    return 0;
}
```

## Manual Setup (without SDK)

Start CloudMock and point any HTTP client at it:

```bash
npx cloudmock &
```

The endpoint `http://localhost:4566` accepts standard AWS Signature V4 requests.
