# CloudMock Ruby SDK

Start a local AWS mock server and get pre-configured clients in your Ruby tests.

## Installation

Add to your Gemfile:

```ruby
gem "cloudmock"
gem "aws-sdk-s3"
gem "aws-sdk-dynamodb"
```

Or install directly:

```sh
gem install cloudmock
```

## Usage

### RSpec

```ruby
require "cloudmock"
require "aws-sdk-s3"
require "aws-sdk-dynamodb"

RSpec.describe "My AWS code" do
  let(:mock) { CloudMock.start }

  after { mock.stop }

  it "creates an S3 bucket" do
    s3 = Aws::S3::Client.new(mock.aws_config)
    s3.create_bucket(bucket: "my-test-bucket")
    resp = s3.list_buckets
    expect(resp.buckets.map(&:name)).to include("my-test-bucket")
  end

  it "puts and gets a DynamoDB item" do
    dynamodb = Aws::DynamoDB::Client.new(mock.aws_config)

    dynamodb.create_table(
      table_name: "users",
      attribute_definitions: [{ attribute_name: "id", attribute_type: "S" }],
      key_schema: [{ attribute_name: "id", key_type: "HASH" }],
      billing_mode: "PAY_PER_REQUEST"
    )

    dynamodb.put_item(table_name: "users", item: { "id" => "u1", "name" => "Alice" })
    resp = dynamodb.get_item(table_name: "users", key: { "id" => "u1" })
    expect(resp.item["name"]).to eq("Alice")
  end
end
```

### Minitest

```ruby
require "minitest/autorun"
require "cloudmock"
require "aws-sdk-s3"

class S3Test < Minitest::Test
  def setup
    @mock = CloudMock.start
    @s3   = Aws::S3::Client.new(@mock.aws_config)
  end

  def teardown
    @mock.stop
  end

  def test_bucket_lifecycle
    @s3.create_bucket(bucket: "test-bucket")
    assert_includes @s3.list_buckets.buckets.map(&:name), "test-bucket"
  end
end
```

### Fixed port

```ruby
mock = CloudMock::Server.new(port: 4566, region: "eu-west-1")
mock.start
```

## Configuration

| Option    | Default      | Description                        |
|-----------|--------------|------------------------------------|
| `port`    | random       | TCP port for the mock server       |
| `region`  | `us-east-1`  | AWS region reported by the server  |
| `profile` | `minimal`    | CloudMock service profile to load  |

## `aws_config` hash

`Server#aws_config` returns a hash suitable for passing directly to any `Aws::*::Client.new`:

```ruby
{
  endpoint:         "http://localhost:<port>",
  region:           "<region>",
  credentials:      Aws::Credentials.new("test", "test"),
  force_path_style: true   # required for S3 path-style access
}
```

## License

BSL-1.1 — see [LICENSE](../../LICENSE).
