// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

// Command terraform-provider-kaiten serves the Kaiten Terraform provider.
package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/kaitencloud/terraform-provider-kaiten/internal/provider"
)

// Format the examples, then generate the Registry documentation from the
// provider schema, the examples and the templates under templates/.
//go:generate terraform fmt -recursive ./examples/
//go:generate go tool tfplugindocs generate --provider-name kaiten

// version is set by GoReleaser at build time.
var version = "dev"

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/kaitencloud/kaiten",
		Debug:   debug,
	}

	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
