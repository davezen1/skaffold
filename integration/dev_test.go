/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"testing"
	"time"

	"github.com/GoogleContainerTools/skaffold/integration/skaffold"
	"github.com/GoogleContainerTools/skaffold/testutil"
	"k8s.io/apimachinery/pkg/util/wait"
)

func TestDev(t *testing.T) {
	var tests = []struct {
		description   string
		trigger       string
		skipFlakyTest bool
	}{
		{
			description: "dev with polling trigger",
			trigger:     "polling",
		},
		{
			description:   "dev with notify trigger",
			trigger:       "notify",
			skipFlakyTest: true,
		},
	}
	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			// TODO(nkubala): fix this test and remove the skip
			if test.skipFlakyTest {
				t.Skip("Skip flaky test")
			}
			if testing.Short() {
				t.Skip("skipping integration test")
			}
			if ShouldRunGCPOnlyTests() {
				t.Skip("skipping test that is not gcp only")
			}

			Run(t, "testdata/dev", "sh", "-c", "echo foo > foo")
			defer Run(t, "testdata/dev", "rm", "foo")

			// Run skaffold build first to fail quickly on a build failure
			skaffold.Build().InDir("testdata/dev").RunOrFail(t)

			ns, client, deleteNs := SetupNamespace(t)
			defer deleteNs()

			stop := skaffold.Dev("--trigger", test.trigger).InDir("testdata/dev").InNs(ns.Name).RunBackground(t)
			defer stop()

			dep := client.GetDeployment("test-dev")

			// Make a change to foo so that dev is forced to delete the Deployment and redeploy
			Run(t, "testdata/dev", "sh", "-c", "echo bar > foo")

			// Make sure the old Deployment and the new Deployment are different
			err := wait.PollImmediate(time.Millisecond*500, 10*time.Minute, func() (bool, error) {
				newDep := client.GetDeployment("test-dev")
				return dep.GetGeneration() != newDep.GetGeneration(), nil
			})
			testutil.CheckError(t, false, err)
		})
	}
}
