// Copyright 2026 KAITEN INC
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
)

func TestFlattenMetadata(t *testing.T) {
	empty := map[string]any{}
	region := map[string]any{"region": "eu-west-1", "tier": float64(2)}

	tests := []struct {
		name     string
		metadata *map[string]any
		prior    jsontypes.Normalized
		want     jsontypes.Normalized
	}{
		{"absent reads as null", nil, jsontypes.NewNormalizedNull(), jsontypes.NewNormalizedNull()},
		{"empty reads as null", &empty, jsontypes.NewNormalizedNull(), jsontypes.NewNormalizedNull()},
		{"empty keeps an equivalent prior", &empty, jsontypes.NewNormalizedValue("{ }"), jsontypes.NewNormalizedValue("{ }")},
		{"set without prior is encoded", &region, jsontypes.NewNormalizedNull(), jsontypes.NewNormalizedValue(`{"region":"eu-west-1","tier":2}`)},
		{"equivalent prior is kept verbatim", &region, jsontypes.NewNormalizedValue(`{"tier": 2, "region": "eu-west-1"}`), jsontypes.NewNormalizedValue(`{"tier": 2, "region": "eu-west-1"}`)},
		{"different prior is replaced", &region, jsontypes.NewNormalizedValue(`{"region":"us-east-1"}`), jsontypes.NewNormalizedValue(`{"region":"eu-west-1","tier":2}`)},
		{"removed metadata reads as null", nil, jsontypes.NewNormalizedValue(`{"region":"us-east-1"}`), jsontypes.NewNormalizedNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := flattenMetadata(tt.metadata, tt.prior)
			if err != nil {
				t.Fatalf("flattenMetadata: %v", err)
			}
			if !got.Equal(tt.want) {
				t.Errorf("got %s, want %s", got, tt.want)
			}
		})
	}
}
