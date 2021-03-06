// +build integration

// Copyright 2020 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	k8Client "github.com/okteto/okteto/pkg/k8s/client"
)

const (
	stackGitRepo   = "git@github.com:okteto/stacks-getting-started.git"
	stackGitFolder = "stacks-getting-started"
	stackManifest  = "okteto-stack.yml"
)

func TestStacks(t *testing.T) {
	if mode == "client" {
		t.Skip("this test is not required for client-side translation")
		return
	}

	ctx := context.Background()
	oktetoPath, err := getOktetoPath(ctx)
	if err != nil {
		t.Fatal(err)
	}

	tName := fmt.Sprintf("TestStacks-%s", runtime.GOOS)
	name := strings.ToLower(fmt.Sprintf("%s-%d", tName, time.Now().Unix()))
	namespace := fmt.Sprintf("%s-%s", name, user)

	t.Run(tName, func(t *testing.T) {
		log.Printf("running %s \n", tName)
		k8Client.Reset()
		if err := createNamespace(ctx, oktetoPath, namespace); err != nil {
			t.Fatal(err)
		}

		log.Printf("created namespace %s \n", namespace)

		if err := cloneGitRepo(ctx, stackGitRepo); err != nil {
			t.Fatal(err)
		}

		log.Printf("cloned repo %s \n", stackGitRepo)

		defer deleteGitRepo(ctx, stackGitFolder)

		if err := deployStack(ctx, oktetoPath, stackManifest); err != nil {
			t.Fatal(err)
		}

		log.Printf("deployed stack using %s \n", stackManifest)

		endpoint := fmt.Sprintf("https://vote-%s.cloud.okteto.net", namespace)
		content, err := getContent(endpoint, 150)
		if err != nil {
			t.Fatalf("failed to get stack content: %s", err)
		}

		if !strings.Contains(content, "Cats vs Dogs!") {
			t.Fatalf("wrong stack content: %s", content)
		}
		if err := destroyStack(ctx, oktetoPath, stackManifest); err != nil {
			t.Fatal(err)
		}

		log.Println("destroyed stack")

		time.Sleep(5 * time.Second)
		_, err = getDeployment(namespace, "vote")
		if err == nil {
			t.Fatalf("'vote' deployment not deleted after 'okteto stack destroy'")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Fatalf("error getting deployment 'vote': %s", err.Error())
		}

		if err := deleteNamespace(ctx, oktetoPath, namespace); err != nil {
			log.Printf("failed to delete namespace %s: %s\n", namespace, err)
		}
	})
}

func cloneGitRepo(ctx context.Context, name string) error {
	log.Printf("cloning git repo %s", name)
	cmd := exec.Command("git", "clone", name)
	o, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("cloning git repo %s failed: %s - %s", name, string(o), err)
	}
	log.Printf("clone git repo %s success", name)
	return nil
}

func deleteGitRepo(ctx context.Context, path string) error {
	log.Printf("delete git repo %s", path)
	err := os.RemoveAll(path)
	if err != nil {
		return fmt.Errorf("delete git repo %s failed: %w", path, err)
	}

	log.Printf("deleted git repo %s", path)
	return nil
}

func deployStack(ctx context.Context, oktetoPath, stackPath string) error {
	log.Printf("okteto stack deploy %s", stackPath)
	cmd := exec.Command(oktetoPath, "stack", "deploy", "-f", stackPath, "--build", "--wait")
	cmd.Env = os.Environ()
	cmd.Dir = stackGitFolder
	o, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("okteto stack deploy failed: %s - %s", string(o), err)
	}
	log.Printf("okteto stack deploy %s success", stackPath)
	return nil
}

func destroyStack(ctx context.Context, oktetoPath, stackManifest string) error {
	log.Printf("okteto stack destroy")
	cmd := exec.Command(oktetoPath, "stack", "destroy", "-f", stackManifest)
	cmd.Env = os.Environ()
	cmd.Dir = stackGitFolder
	o, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("okteto stack destroy failed: %s - %s", string(o), err)
	}
	log.Printf("okteto stack destroy success")
	return nil
}
