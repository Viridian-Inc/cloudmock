terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  access_key                  = "test"
  secret_key                  = "test"
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    s3       = "http://localhost:4566"
    dynamodb = "http://localhost:4566"
    sqs      = "http://localhost:4566"
    sns      = "http://localhost:4566"
    lambda   = "http://localhost:4566"
    iam      = "http://localhost:4566"
  }
}

resource "aws_s3_bucket" "example" {
  bucket = "my-example-bucket"
}

resource "aws_dynamodb_table" "example" {
  name         = "my-example-table"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"

  attribute {
    name = "pk"
    type = "S"
  }
}

resource "aws_sqs_queue" "example" {
  name = "my-example-queue"
}
