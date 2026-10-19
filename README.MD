# CS6650 HW5 – Product API on AWS ECS

This project deploys a Go-based Product API to AWS ECS using Terraform. Terraform provisions the required AWS infrastructure, builds the Docker image, and pushes the image to Amazon ECR.

## Architecture

The application uses the following components:

- **Go and Gin:** REST API implementation.
- **Docker:** Containerizes the Product API.
- **Amazon ECR:** Stores the Docker image.
- **Amazon ECS:** Runs the containerized application.
- **Terraform:** Provisions and manages the infrastructure.
- **Amazon CloudWatch Logs:** Collects application logs.
- **Prometheus:** Exposes application metrics through `/metrics`.

## Prerequisites

Install and configure the following:

- AWS CLI
- Terraform
- Docker Desktop
- Git

An active AWS Academy Learner Lab session with the required permissions is also needed.

## 1. Configure AWS Credentials

Start your AWS Academy Learner Lab and retrieve your temporary AWS credentials.

Configure the AWS CLI:

```powershell
aws configure
```

Enter the access key ID, secret access key, and region when prompted. Use `us-east-1` for this deployment.

Configure the temporary session token:

```powershell
aws configure set aws_session_token "<YOUR-TEMP-SESSION-TOKEN>"
```

Do not commit AWS credentials or session tokens to GitHub.

## 2. Deploy the Infrastructure

Open a terminal in the repository root and navigate to the Terraform directory:

```powershell
cd terraform
terraform init
terraform validate
terraform plan
terraform apply
```

Review the Terraform plan and type `yes` when prompted.

Terraform creates the required infrastructure and builds and pushes the Product API Docker image to Amazon ECR.

After a successful deployment, Terraform displays the ECS cluster and service names.

## 3. Retrieve the Public IP Address

Run these commands from the `terraform` directory in PowerShell:

```powershell
$cluster = terraform output -raw ecs_cluster_name
$service = terraform output -raw ecs_service_name

$task = aws ecs list-tasks `
    --cluster $cluster `
    --service-name $service `
    --region us-east-1 `
    --query "taskArns[0]" `
    --output text

$eni = aws ecs describe-tasks `
    --cluster $cluster `
    --tasks $task `
    --region us-east-1 `
    --query "tasks[0].attachments[0].details[?name=='networkInterfaceId'].value | [0]" `
    --output text

$publicIp = aws ec2 describe-network-interfaces `
    --network-interface-ids $eni `
    --region us-east-1 `
    --query "NetworkInterfaces[0].Association.PublicIp" `
    --output text

Write-Output $publicIp
```

Use the returned public IP address for the API requests below.

## 4. Test the Product API

Replace `<PUBLIC-IP>` in each command with the public IP address returned in the previous step.

### GET a Product

```powershell
curl.exe -i "http://<PUBLIC-IP>:8080/products/12345"
```

Expected result: `HTTP/1.1 200 OK`, followed by the product's JSON data.

### POST Product Details

The request body must contain all required fields, and `product_id` must match the ID in the URL.

```powershell
$body = @{
    product_id = 12345
    sku = "ABC-123-XYZ"
    manufacturer = "Acme Corporation"
    category_id = 456
    weight = 1250
    some_other_id = 789
} | ConvertTo-Json -Compress

Invoke-WebRequest -UseBasicParsing `
    -Uri "http://<PUBLIC-IP>:8080/products/12345/details" `
    -Method Post `
    -ContentType "application/json" `
    -Body $body
```

Expected result: HTTP `204 No Content`.

### GET Metrics

```powershell
curl.exe -i "http://<PUBLIC-IP>:8080/metrics"
```

Expected result: `HTTP/1.1 200 OK`, followed by Prometheus metrics, including Go runtime metrics and Product API request metrics.

## 5. Verify the ECS Service

Check that the ECS service is active and the desired number of tasks is running:

```powershell
aws ecs describe-services `
    --cluster hw5-product-api-fork-cluster `
    --services hw5-product-api-fork `
    --region us-east-1 `
    --query "services[0].{Status:status,Running:runningCount,Desired:desiredCount}" `
    --output json
```

The service should be active, with its running task count matching its desired task count.

## 6. Clean Up

When you have finished testing, remove the AWS resources created by Terraform.

Run this command from the `terraform` directory:

```powershell
terraform destroy
```

Review the proposed changes and type `yes` to confirm.

Destroying the infrastructure stops the deployed service and removes the resources managed by this Terraform configuration.

## Notes

- The API listens on port `8080`.
- The example product ID is `12345`.
- Product data is stored in an in-memory Go map; changes are not persisted to a database and can be lost when the application restarts.
- AWS Academy credentials are temporary and may need to be refreshed when the Learner Lab session expires.