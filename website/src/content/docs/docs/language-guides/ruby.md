---
title: Ruby
description: Using CloudMock with the AWS SDK for Ruby
---

# Ruby

## CloudMock Ruby SDK

The CloudMock Ruby SDK auto-manages the server lifecycle:

```ruby
require "cloudmock"

cm = CloudMock.start
s3 = Aws::S3::Client.new(cm.aws_config)
# ... use s3 client
cm.stop
```

Install: `gem install cloudmock`

## Manual Setup

```ruby
require "aws-sdk-s3"
require "aws-sdk-dynamodb"

Aws.config.update(
  endpoint: "http://localhost:4566",
  region: "us-east-1",
  credentials: Aws::Credentials.new("test", "test")
)

s3 = Aws::S3::Client.new(force_path_style: true)
s3.create_bucket(bucket: "test-bucket")
s3.put_object(bucket: "test-bucket", key: "hello.txt", body: "world")
```

## DynamoDB Example

```ruby
ddb = Aws::DynamoDB::Client.new

ddb.create_table(
  table_name: "users",
  key_schema: [{ attribute_name: "pk", key_type: "HASH" }],
  attribute_definitions: [{ attribute_name: "pk", attribute_type: "S" }],
  billing_mode: "PAY_PER_REQUEST"
)

ddb.put_item(
  table_name: "users",
  item: { "pk" => "user-1", "name" => "Alice" }
)
```

## Testing with Minitest

```ruby
require "minitest/autorun"
require "cloudmock"
require "aws-sdk-s3"

class AWSTest < Minitest::Test
  def setup
    @cm = CloudMock.start
    @s3 = Aws::S3::Client.new(@cm.aws_config)
  end

  def teardown
    @cm.stop
  end

  def test_s3_bucket
    @s3.create_bucket(bucket: "test")
    resp = @s3.list_buckets
    assert resp.buckets.any? { |b| b.name == "test" }
  end
end
```

## Testing with RSpec

```ruby
require "cloudmock"

RSpec.configure do |config|
  config.before(:suite) do
    $cm = CloudMock.start
  end

  config.after(:suite) do
    $cm.stop
  end
end

RSpec.describe "S3" do
  let(:s3) { Aws::S3::Client.new($cm.aws_config) }

  it "creates a bucket" do
    s3.create_bucket(bucket: "test")
    expect(s3.list_buckets.buckets.map(&:name)).to include("test")
  end
end
```
