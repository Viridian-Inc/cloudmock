resource "aws_s3_bucket" "data" {
  bucket = "my-app-data"
}

resource "aws_s3_bucket_versioning" "data" {
  bucket = aws_s3_bucket.data.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_dynamodb_table" "items" {
  name         = "items"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  tags = {
    Environment = "local"
  }
}

resource "aws_sqs_queue" "jobs" {
  name                       = "jobs"
  visibility_timeout_seconds = 30
  message_retention_seconds  = 86400
}

resource "aws_sqs_queue" "jobs_dlq" {
  name = "jobs-dlq"
}

resource "aws_sqs_queue_redrive_policy" "jobs" {
  queue_url = aws_sqs_queue.jobs.id
  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.jobs_dlq.arn
    maxReceiveCount     = 3
  })
}

output "bucket_name" {
  value = aws_s3_bucket.data.id
}

output "dynamodb_table" {
  value = aws_dynamodb_table.items.name
}

output "queue_url" {
  value = aws_sqs_queue.jobs.url
}
