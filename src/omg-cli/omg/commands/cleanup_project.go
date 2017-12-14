/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package commands

import (
	"context"
	"fmt"
	"log"

	"omg-cli/config"
	"omg-cli/google"

	"github.com/alecthomas/kingpin"
	googleauth "golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
)

type CleanupProjectCommand struct {
	logger         *log.Logger
	envDir         string
	envCfg         *config.EnvConfig
	cfg            *config.Config
	cleanupService google.CleanupService
	dryRun         bool
}

const CleanupProjectName = "cleanup-project"

func (cmd *CleanupProjectCommand) register(app *kingpin.Application) {
	c := app.Command(CleanupProjectName, "Delete VMs created by Ops Manager upgrades and abandoned by BOSH").Action(cmd.run)
	registerEnvConfigFlag(c, &cmd.envDir)
	c.Flag("dry-run", "view deletion plan, don't perform it").Default("true").BoolVar(&cmd.dryRun)
}

func (cmd *CleanupProjectCommand) parseArgs() {
	var err error
	cmd.envCfg, err = config.ConfigFromEnvDirectory(cmd.envDir)
	if err != nil {
		cmd.logger.Fatalf("loading environment config: %v", err)
	}
	cmd.cfg, err = config.TerraformFromEnvDirectory(cmd.envDir)
	if err != nil {
		cmd.logger.Fatalf("loading environment config: %v", err)
	}

	gcpClient, err := googleauth.DefaultClient(context.Background(), compute.CloudPlatformScope)
	if err != nil {
		cmd.logger.Fatalf("loading application default credentials: %v.\nHave you ran `gcloud auth application-default login`?", err)
	}

	cmd.cleanupService, err = google.NewCleanupService(cmd.logger, cmd.envCfg.ProjectID, gcpClient, cmd.dryRun)
	if err != nil {
		cmd.logger.Fatalf("creating CleanupService: %v", err)
	}
	return
}

func (cmd *CleanupProjectCommand) run(c *kingpin.ParseContext) error {
	cmd.parseArgs()

	steps := []step{
		cmd.deleteUpgradedOpsManagers,
		cmd.deleteErtVMs,
		cmd.deleteServicesVMs,
		cmd.deleteDirectorVM,
	}

	return run(steps)
}

// Delete Ops Manager VMs created by the C0 Pipeline
// These VMs are identifiable by <original vm name>-<date of upgrade>
func (cmd *CleanupProjectCommand) deleteUpgradedOpsManagers() error {
	deleted, err := cmd.cleanupService.DeleteVM(google.WithSubNetwork(cmd.cfg.MgmtSubnetName),
		google.WithTag(fmt.Sprintf("%s-ops-manager", cmd.envCfg.EnvName)),
		google.WithNameRegex(fmt.Sprintf("%s-ops-manager-.*", cmd.envCfg.EnvName)))
	cmd.logger.Printf("deleteUpgradedOpsManagers: deleted %d VMs", deleted)

	return err
}

// Delete ERT VMs created by BOSH
func (cmd *CleanupProjectCommand) deleteErtVMs() error {
	deleted, err := cmd.cleanupService.DeleteVM(google.WithSubNetwork(cmd.cfg.ErtSubnetName),
		google.WithTag("p-bosh"),
		google.WithNameRegex("vm-.*"))
	cmd.logger.Printf("deleteErtVMs: deleted %d VMs", deleted)
	return err
}

// Delete Services VMs created by BOSH
func (cmd *CleanupProjectCommand) deleteServicesVMs() error {
	deleted, err := cmd.cleanupService.DeleteVM(google.WithSubNetwork(cmd.cfg.ServicesSubnetName),
		google.WithTag("p-bosh"),
		google.WithNameRegex("vm-.*"))
	cmd.logger.Printf("deleteServicesVMs: deleted %d VMs", deleted)
	return err
}

// Delete BOSH director VM
func (cmd *CleanupProjectCommand) deleteDirectorVM() error {
	deleted, err := cmd.cleanupService.DeleteVM(google.WithSubNetwork(cmd.cfg.MgmtSubnetName),
		google.WithLabel("job", "bosh"),
		google.WithNameRegex("vm-.*"))
	cmd.logger.Printf("deleteDirectorVM: deleted %d VMs", deleted)

	return err
}
