terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

locals {
  project_name = "aws-serverless-http-response-streaming"
}

provider "aws" {}

data "aws_iam_policy_document" "assume_role" {
  statement {
    effect = "Allow"

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }

    actions = ["sts:AssumeRole"]
  }
}

data "archive_file" "lambda" {
  type        = "zip"
  source_file = "bootstrap"
  output_path = "bootstrap.zip"
}

resource "aws_iam_role" "lambda_role" {
  name               = "${local.project_name}-lambda-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role.json
}


resource "aws_lambda_function" "main" {
  function_name = local.project_name

  filename = "bootstrap.zip"
  handler  = "./bootstrap"
  # runtime          = "go1.x"
  runtime          = "provided"
  source_code_hash = data.archive_file.lambda.output_base64sha256

  role = aws_iam_role.lambda_role.arn
}

resource "aws_lambda_function_url" "main" {
  function_name      = aws_lambda_function.main.function_name
  authorization_type = "NONE"
  invoke_mode        = "RESPONSE_STREAM"
}

output "lambda_function_url" {
  value = aws_lambda_function_url.main.function_url
}
