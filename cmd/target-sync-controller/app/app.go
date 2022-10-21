// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	"fmt"
	"github.com/gardener/landscaper-service/pkg/apis/config"
	"k8s.io/utils/pointer"
	"os"

	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	lsinstall "github.com/gardener/landscaper/apis/core/install"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	lssinstall "github.com/gardener/landscaper-service/pkg/apis/core/install"
	"github.com/gardener/landscaper-service/pkg/controllers/targetsync"
	"github.com/gardener/landscaper-service/pkg/crdmanager"
	"github.com/gardener/landscaper-service/pkg/version"
)

// NewTargetSyncControllerCommand creates a new command for the landscaper service controller
func NewTargetSyncControllerCommand(ctx context.Context) *cobra.Command {
	options := NewOptions()

	cmd := &cobra.Command{
		Use:   "target-sync-controller",
		Short: "Target sync controller syncronises secrets into custom namespaces and creates corresponding targets",

		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(ctx); err != nil {
				fmt.Print(err)
				os.Exit(1)
			}
			if err := options.run(ctx); err != nil {
				options.Log.Error(err, "unable to run target sync controller")
			}
		},
	}

	options.AddFlags(cmd.Flags())

	return cmd
}

func (o *options) run(ctx context.Context) error {
	o.Log.Info(fmt.Sprintf("Start Target Sync Controller with version %q", version.Get().String()))

	opts := manager.Options{
		LeaderElection: false,
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), opts)
	if err != nil {
		return fmt.Errorf("unable to setup manager: %w", err)
	}

	if err := o.ensureCRDs(ctx, mgr); err != nil {
		return err
	}

	lssinstall.Install(mgr.GetScheme())
	lsinstall.Install(mgr.GetScheme())

	ctrlLogger := o.Log.WithName("controllers")
	if err := targetsync.AddControllerToManager(ctrlLogger, mgr); err != nil {
		return fmt.Errorf("unable to setup landscaper deployments controller: %w", err)
	}

	o.Log.Info("starting the controllers")
	if err := mgr.Start(ctx); err != nil {
		o.Log.Error(err, "error while running manager")
		os.Exit(1)
	}
	return nil
}

func (o *options) ensureCRDs(ctx context.Context, mgr manager.Manager) error {
	ctx = logging.NewContext(ctx, logging.Wrap(ctrl.Log.WithName("crdManager")))

	crdConfig := config.CrdManagementConfiguration{
		DeployCustomResourceDefinitions: pointer.Bool(true),
		ForceUpdate:                     pointer.Bool(true),
	}
	crdmgr, err := crdmanager.NewCrdManager(mgr, crdConfig)
	if err != nil {
		return fmt.Errorf("unable to setup CRD manager: %w", err)
	}

	if err := crdmgr.EnsureCRDs(ctx); err != nil {
		return fmt.Errorf("failed to handle CRDs: %w", err)
	}

	return nil
}
