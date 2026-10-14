variable "kaiten_auth_token" {
  type      = string
  sensitive = true
}

provider "kaiten" {
  base_url   = "https://kaiten.example.com/api"
  auth_token = var.kaiten_auth_token
}
