import * as cdk from "aws-cdk-lib";
import * as s3 from "aws-cdk-lib/aws-s3";
import * as dynamodb from "aws-cdk-lib/aws-dynamodb";
import * as sqs from "aws-cdk-lib/aws-sqs";
import { Construct } from "constructs";

export class ExampleStack extends cdk.Stack {
  constructor(scope: Construct, id: string) {
    super(scope, id);
    new s3.Bucket(this, "ExampleBucket");
    new dynamodb.Table(this, "ExampleTable", {
      partitionKey: { name: "pk", type: dynamodb.AttributeType.STRING },
      billingMode: dynamodb.BillingMode.PAY_PER_REQUEST,
    });
    new sqs.Queue(this, "ExampleQueue");
  }
}
