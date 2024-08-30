// Copyright 2022 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package autokey_example

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"testing"

	"github.com/GoogleCloudPlatform/cloud-foundation-toolkit/infra/blueprint-test/pkg/gcloud"
	"github.com/GoogleCloudPlatform/cloud-foundation-toolkit/infra/blueprint-test/pkg/tft"
	"github.com/GoogleCloudPlatform/cloud-foundation-toolkit/infra/blueprint-test/pkg/utils"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2/google"
)

func validateKeyHandleVersion(input string, projectId string, location string) bool {
	pattern := fmt.Sprintf(`^projects/%s/locations/%s/keyRings/autokey/cryptoKeys/.*-(bigquery-dataset|compute-disk|storage-bucket)-.*?/cryptoKeyVersions/1$`, projectId, location)
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(input)
}

func TestAutokeyExample(t *testing.T) {
	bpt := tft.NewTFBlueprintTest(t)
	bpt.DefineVerify(func(assert *assert.Assertions) {
		bpt.DefaultVerify(assert)

		projectId := bpt.GetStringOutput("autokey_project_id")
		autokeyConfig := bpt.GetStringOutput("autokey_config_id")
		location := bpt.GetStringOutput("location")

		autokeyConfigUrl := fmt.Sprintf("https://cloudkms.googleapis.com/v1/%s", autokeyConfig)

		httpClient, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/cloud-platform")

		if err != nil {
			return
		}

		resp, err := httpClient.Get(autokeyConfigUrl)
		if err != nil {
			return
		}

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		result := utils.ParseJSONResult(t, string(body))

		// Asserting if Autokey configuration was created
		autokeyConfigProject := result.Get("keyProject").String()
		assert.Equal(autokeyConfigProject, fmt.Sprintf("projects/%s", projectId), "autokey expected for project %s", projectId)

		// Asserting if Autokey keyring was created
		op := gcloud.Runf(t, "--project=%s kms keyrings list --location %s --filter name:autokey", projectId, location).Array()[0].Get("name")
		assert.Contains(op.String(), fmt.Sprintf("projects/%s/locations/%s/keyRings/autokey", projectId, location), "Contains Autokey KeyRing")

		// Asserting if Autokey keyHandles were created
		op1 := gcloud.Runf(t, "kms keys list --project=%s --keyring autokey --location %s", projectId, location).Array()
		for _, element := range op1 {
			assert.True(validateKeyHandleVersion(element.Get("primary").Map()["name"].Str, projectId, location), "Contains KeyHandles")
		}
	})

	bpt.Test()
}
