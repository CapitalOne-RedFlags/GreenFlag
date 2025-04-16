# **AWS SAM Configuration and Template Guide**

## **Overview**
This guide provides an explanation of the `samconfig.toml` and `template.yaml` files found in the `deployments/` directory. These files are crucial for configuring and deploying AWS resources using the AWS Serverless Application Model (SAM).

---

## **`samconfig.toml`**

### **Purpose**
The `samconfig.toml` file defines deployment settings and parameters for AWS SAM. It ensures consistency across deployments by specifying stack configurations, regions, IAM permissions, and parameter overrides.

### **File Path**
`deployments/samconfig.toml`

### **Key Sections**
#### **Deployment Configuration (`default.deploy`)**
```toml
[default.deploy]
[default.deploy.parameters]
stack_name = "TransactionConsumerStack"
region = "us-east-1"
confirm_changeset = true
capabilities = "CAPABILITY_IAM"
```
- `stack_name`: Defines the CloudFormation stack name.
- `region`: AWS region where the stack is deployed.
- `confirm_changeset`: If `true`, prompts before applying changes.
- `capabilities`: Grants IAM role creation permissions.

#### **Parameter Overrides**
```toml
parameter_overrides = [
  "TransactionQueueARN=arn:aws:sqs:us-east-1:140023383737:Bank_Transactions",
  "TransactionDLQARN=arn:aws:sqs:us-east-1:140023383737:TransactionDLQ",
  "DynamoDBTableName=Transactions",
  "AWSRegion=us-east-1"
]
```
- These parameters allow dynamic configuration without modifying `template.yaml`.
- The values here override the default parameters defined in the template.

#### **Sync Configuration (`default.sync`)**
```toml
[default.sync]
[default.sync.parameters]
stack_name = "TransactionConsumerStack"
watch = true
region = "us-east-1"
profile = "AdministratorAccess-140023383737"
```
- Enables `sam sync`, which updates the stack in real-time as files change.
- `watch = true` keeps the deployment up to date.
- The `profile` specifies the AWS credential profile for authentication.

---

## **`template.yaml`**

### **Purpose**
The `template.yaml` file is an AWS CloudFormation template written using AWS Serverless Application Model (AWS SAM). It defines the infrastructure required for a transaction processing system, including AWS Lambda functions, an SQS queue, a DynamoDB table, and an SNS topic for fraud alerts.

### **File Path**
`deployments/template.yaml`

### **Key Components**

#### **Global Configuration**
```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: AWS::Serverless-2016-10-31
```
- Specifies the AWS CloudFormation version.
- Uses the AWS Serverless Application Model (SAM) transform to simplify serverless application deployment.

#### **Parameters**
```yaml
Parameters:
  TransactionQueueARN:
    Type: String
    Description: ARN of the SQS queue holding transaction messages
  
  DynamoDBTableName:
    Type: String
    Description: Name of the DynamoDB table
    Default: Transactions

  DynamoDBEndpoint:
    Type: String
    Description: DynamoDB endpoint (useful for local testing)
    Default: https://dynamodb.us-east-1.amazonaws.com

  AWSRegion:
    Type: String
    Description: AWS region for resource deployment
    Default: us-east-1
```
- Defines configurable parameters for flexibility in deployment.
- Allows users to specify an existing SQS queue, DynamoDB table name, and AWS region.

#### **Resources**

##### **1. DynamoDB Table**
```yaml
TransactionsTable:
  Type: AWS::DynamoDB::Table
  Properties:
    TableName: !Ref DynamoDBTableName
    AttributeDefinitions:
      - AttributeName: AccountID
        AttributeType: S
      - AttributeName: TransactionID
        AttributeType: S
    KeySchema:
      - AttributeName: AccountID
        KeyType: HASH
      - AttributeName: TransactionID
        KeyType: RANGE
    BillingMode: PAY_PER_REQUEST
    StreamSpecification:
      StreamViewType: NEW_AND_OLD_IMAGES
```
- Defines a DynamoDB table to store transaction data.
- Uses `PAY_PER_REQUEST` billing mode for automatic scaling.
- Enables DynamoDB Streams to capture real-time changes.

##### **2. SNS Topic for Fraud Alerts**
```yaml
NotificationTopic:
  Type: AWS::SNS::Topic
  Properties:
    TopicName: FraudAlerts
```
- Creates an SNS topic for publishing fraud alerts detected in transactions.

##### **3. Transaction Processing Lambda Function**
```yaml
TransactionPipelineFunction:
  Type: AWS::Serverless::Function
  Properties:
    FunctionName: TransactionPipelineFunction
    CodeUri: ../
    Handler: bootstrap
    Runtime: provided.al2
    Environment:
      Variables:
        DYNAMODB_TABLE_NAME: !Ref DynamoDBTableName
    Policies:
      - AWSLambdaBasicExecutionRole
      - Statement:
          - Effect: Allow
            Action:
              - dynamodb:PutItem
              - dynamodb:UpdateItem
              - dynamodb:GetItem
              - dynamodb:DescribeTable
            Resource: !GetAtt TransactionsTable.Arn
          - Effect: Allow
            Action:
              - sqs:ReceiveMessage
              - sqs:DeleteMessage
              - sqs:GetQueueAttributes
            Resource: !Ref TransactionQueueARN
    Events:
      SQSEvent:
        Type: SQS
        Properties:
          Queue: !Ref TransactionQueueARN
          BatchSize: 10
          MaximumBatchingWindowInSeconds: 5
```
- This Lambda function processes transactions by reading messages from the SQS queue and updating DynamoDB.
- **`CodeUri: ../`** indicates that the Lambda function’s executable code is located in the `parent-directory/.aws-sam.`. This can be adjusted to point to the correct deployment package location.
- **`Handler:`** bootstrap is necessary because AWS Lambda no longer supports go1.x, requiring a custom bootstrap executable for Go applications. The bootstrap file is the compiled Go binary and must be placed at the artifacts directory under `.aws-sam` where `aws sam` will look for executables (specified in CodeUri). The makefile in project root generates these executables for you when you run `make build` or `make sync`.
- Uses `provided.al2` runtime (Amazon Linux 2) for custom compiled executables.
- Grants necessary permissions for SQS and DynamoDB interactions.

##### **4. Fraud Detection Lambda Function**
```yaml
FraudPipelineFunction:
  Type: AWS::Serverless::Function
  Properties:
    FunctionName: FraudPipelineFunction
    CodeUri: ../
    Handler: bootstrap
    Runtime: provided.al2
    Policies:
      - AWSLambdaBasicExecutionRole
      - Statement:
          - Effect: Allow
            Action:
              - sns:Publish
            Resource: !Ref NotificationTopic
    Events:
      DynamoDBStream:
        Type: DynamoDB
        Properties:
          Stream: !GetAtt TransactionsTable.StreamArn
          StartingPosition: TRIM_HORIZON
          BatchSize: 10
          MaximumBatchingWindowInSeconds: 5
```
- This Lambda function detects suspicious transactions using DynamoDB Streams and publishes alerts to SNS.
- **`CodeUri: ../`** means the function’s executable code is in the `parent-directory/.aws-sam.`
- **`Handler:`** bootstrap is necessary because AWS Lambda no longer supports go1.x, requiring a custom bootstrap executable for Go applications. The bootstrap file is the compiled Go binary and must be placed at the artifacts directory under `.aws-sam` where `aws sam` will look for executables (specified in CodeUri). The makefile in project root generates these executables for you when you run `make build` or `make sync`.
- Uses `provided.al2` runtime for compatibility with compiled applications.
- Grants permission to publish to the SNS topic.

#### **Outputs**
```yaml
Outputs:
  DynamoDBTableNameOut:
    Description: "Name of the DynamoDB table"
    Value: !Ref TransactionsTable

  NotificationTopicArn:
    Description: "ARN of the SNS topic"
    Value: !Ref NotificationTopic

  TransactionPipelineArn:
    Description: "ARN of the TransactionPipelineFunction"
    Value: !GetAtt TransactionPipelineFunction.Arn

  FraudPipelineArn:
    Description: "ARN of the FraudPipelineFunction"
    Value: !GetAtt FraudPipelineFunction.Arn
```
- Provides references to key AWS resources after deployment, making it easier to integrate with other services.

---

#### **Parameters vs Outputs**

##### **Parameters**
- **Purpose:** Allow users to **customize** the CloudFormation/SAM stack during deployment.
- **Usage:** Used for **input values** that can change between deployments.
- **Defined in:** The `Parameters` section of `template.yaml`.
- **Example Use Case:** Allowing users to specify an **SQS queue ARN**, a **DynamoDB table name**, or an **AWS region** at deployment time.

**Example:**
```yaml
Parameters:
  DynamoDBTableName:
    Type: String
    Description: Name of the DynamoDB table
    Default: Transactions
```
- Here, `DynamoDBTableName` is a parameter that allows the user to **set the table name dynamically**.
- If the user does not provide a value, it defaults to `"Transactions"`.

##### **Outputs**
- **Purpose:** Provide useful **information about deployed resources** after the stack is created.
- **Usage:** Outputs can be used in scripts, cross-stack references, or displayed to the user after deployment.
- **Defined in:** The `Outputs` section of `template.yaml`.
- **Example Use Case:** Returning the **ARN of a Lambda function**, **SNS topic**, or **DynamoDB table name**.

**Example:**
```yaml
Outputs:
  TransactionPipelineArn:
    Description: "ARN of the TransactionPipelineFunction"
    Value: !GetAtt TransactionPipelineFunction.Arn
```
- This outputs the **ARN** of the `TransactionPipelineFunction` after deployment.
- This ARN can be used by other AWS services, infrastructure scripts, or developers.

##### **Key Differences**
| Feature    | Parameters  | Outputs  |
|------------|------------|----------|
| **Purpose** | Accepts user-defined input values for customization | Displays or exports key information about deployed resources |
| **Defined in** | `Parameters` section | `Outputs` section |
| **Usage** | Used **before** deployment (user-defined) | Generated **after** deployment (system-defined) |
| **Example** | Let users specify a **DynamoDB table name** | Output the **DynamoDB table ARN** |
| **Modifiable after deployment?** | Yes (by redeploying with new values) | No (reflects current deployed state) |

---
## **Conclusion**
- `template.yaml` defines a serverless architecture for processing transactions and detecting fraud.
- `CodeUri` specifies where the Lambda function source code is located. By default, it points to `../`, meaning the function’s deployment package is outside the `deployments/` directory.
- This template enables automated, scalable, event-driven transaction processing using AWS services.

