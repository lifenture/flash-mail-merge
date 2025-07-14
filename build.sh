#!/bin/bash

# Define the output ZIP file name
deploy_package="flash-mail-merge-lambda.zip"

# Clean up any previous build artifacts
if [ -f "$deploy_package" ]; then
    rm "$deploy_package"
fi

# Build the Go binary for Lambda (must be named 'bootstrap')
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap

# Create a deployment package
zip "$deploy_package" bootstrap

# Clean up the Go binary
rm bootstrap

echo "Deployment package '$deploy_package' created successfully."
