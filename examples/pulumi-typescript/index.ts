import * as aws from "@pulumi/aws";

const bucket = new aws.s3.Bucket("example-bucket");
const table = new aws.dynamodb.Table("example-table", {
    attributes: [{ name: "pk", type: "S" }],
    hashKey: "pk",
    billingMode: "PAY_PER_REQUEST",
});
const queue = new aws.sqs.Queue("example-queue");

export const bucketName = bucket.id;
export const tableName = table.name;
export const queueUrl = queue.url;
