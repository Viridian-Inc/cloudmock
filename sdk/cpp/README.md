# CloudMock C/C++ SDK

Local AWS emulation for C/C++ tests.

## Build

```bash
make          # builds libcloudmock.a
make test     # runs tests
```

## Usage (C)

```c
#include <cloudmock.h>

int main() {
    cloudmock_t *cm = cloudmock_start(NULL);
    printf("Endpoint: %s\n", cloudmock_endpoint(cm));

    /* Use AWS SDK for C++ with endpoint: cloudmock_endpoint(cm) */

    cloudmock_stop(cm);
    return 0;
}
```

## Usage (C++)

```cpp
#include <cloudmock.h>
#include <aws/core/Aws.h>
#include <aws/s3/S3Client.h>

int main() {
    auto *cm = cloudmock_start(nullptr);

    Aws::SDKOptions options;
    Aws::InitAPI(options);
    {
        Aws::Client::ClientConfiguration config;
        config.endpointOverride = cloudmock_endpoint(cm);
        config.scheme = Aws::Http::Scheme::HTTP;
        config.region = "us-east-1";

        Aws::S3::S3Client s3(config);
        // Use s3...
    }
    Aws::ShutdownAPI(options);

    cloudmock_stop(cm);
}
```
