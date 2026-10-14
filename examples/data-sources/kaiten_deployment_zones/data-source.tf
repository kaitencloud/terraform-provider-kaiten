data "kaiten_deployment_zones" "all" {}

locals {
  production_zone_slugs = [
    for zone in data.kaiten_deployment_zones.all.deployment_zones : zone.slug
    if zone.type == "production"
  ]
}
