resource "kaiten_deployment_zone" "eu_west" {
  name        = "AWS eu-west-1"
  slug        = "aws-eu-west-1"
  type        = "production"
  description = "Primary production region for EU customers."

  metadata = jsonencode({
    region = "eu-west-1"
  })
}
