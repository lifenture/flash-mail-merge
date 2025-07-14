.PHONY: build package deploy clean test help

# Default target
.DEFAULT_GOAL := help

# Variables
TEMPLATE_FILE := deploy/template.yaml
PACKAGED_TEMPLATE := packaged.yaml
STACK_NAME := flash-mail-merge
S3_BUCKET := flash-mail-merge-deployment-$(shell aws sts get-caller-identity --query Account --output text)
REGION := $(shell aws configure get region)
STAGE := dev

# Build the Go binary for Lambda
build:
	@echo "Building Go binary for Lambda..."
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap main.go
	@echo "Build complete!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bootstrap
	rm -f $(PACKAGED_TEMPLATE)
	rm -rf .aws-sam/
	@echo "Clean complete!"

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Package the SAM application
package: build
	@echo "Packaging SAM application..."
	@echo "Using S3 bucket: $(S3_BUCKET)"
	@echo "Using region: $(REGION)"
	sam package \
		--template-file $(TEMPLATE_FILE) \
		--output-template-file $(PACKAGED_TEMPLATE) \
		--s3-bucket $(S3_BUCKET) \
		--region $(REGION)
	@echo "Package complete!"

# Deploy the SAM application
deploy: package
	@echo "Deploying SAM application..."
	sam deploy \
		--template-file $(PACKAGED_TEMPLATE) \
		--stack-name $(STACK_NAME) \
		--capabilities CAPABILITY_IAM \
		--parameter-overrides Stage=$(STAGE) \
		--region $(REGION) \
		--no-fail-on-empty-changeset
	@echo "Deploy complete!"

# Build and deploy in one command
deploy-all: build package deploy

# Validate the SAM template
validate:
	@echo "Validating SAM template..."
	sam validate --template $(TEMPLATE_FILE)
	@echo "Template is valid!"

# Start local API for testing
local:
	@echo "Starting local API..."
	sam local start-api --template $(TEMPLATE_FILE)

# Create S3 bucket for deployment if it doesn't exist
create-bucket:
	@echo "Creating deployment bucket if it doesn't exist..."
	@aws s3api head-bucket --bucket $(S3_BUCKET) 2>/dev/null || \
		aws s3api create-bucket --bucket $(S3_BUCKET) --region $(REGION) \
		$(shell [ "$(REGION)" != "us-east-1" ] && echo "--create-bucket-configuration LocationConstraint=$(REGION)")
	@echo "Bucket ready!"

# Delete the stack
delete:
	@echo "Deleting stack..."
	aws cloudformation delete-stack --stack-name $(STACK_NAME) --region $(REGION)
	@echo "Stack deletion initiated!"

# Show help
help:
	@echo "Available targets:"
	@echo "  build        - Build the Go binary for Lambda"
	@echo "  test         - Run tests"
	@echo "  package      - Package the SAM application"
	@echo "  deploy       - Deploy the SAM application"
	@echo "  deploy-all   - Build, package, and deploy in one command"
	@echo "  validate     - Validate the SAM template"
	@echo "  local        - Start local API for testing"
	@echo "  create-bucket- Create S3 bucket for deployment"
	@echo "  clean        - Clean build artifacts"
	@echo "  delete       - Delete the CloudFormation stack"
	@echo "  help         - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  STACK_NAME   - CloudFormation stack name (default: $(STACK_NAME))"
	@echo "  STAGE        - Deployment stage (default: $(STAGE))"
	@echo "  REGION       - AWS region (default: $(REGION))"
