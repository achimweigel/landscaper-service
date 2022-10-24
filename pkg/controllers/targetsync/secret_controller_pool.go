// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"
	"fmt"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/gardener/landscaper-service/pkg/apis/core/v1alpha1"
)

var secretControllerPoolInstance = newSecretControllerPool()

func getSecretControllerPool() *secretControllerPool {
	return secretControllerPoolInstance
}

type secretControllerPool struct {
	rootContext context.Context
	items       map[client.ObjectKey]*secretControllerItem
}

type secretControllerItem struct {
	cancel context.CancelFunc
}

func newSecretControllerPool() *secretControllerPool {
	return &secretControllerPool{
		rootContext: context.Background(),
		items:       map[client.ObjectKey]*secretControllerItem{},
	}
}

func (p *secretControllerPool) containsItem(targetSyncKey client.ObjectKey) bool {
	_, exists := p.items[targetSyncKey]
	return exists
}

func (p *secretControllerPool) deleteItem(targetSyncKey client.ObjectKey) {
	delete(p.items, targetSyncKey)
}

func (p *secretControllerPool) createSecretController(ctx context.Context, targetClient client.Client, targetSync *v1alpha1.TargetSync) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{})

	targetSyncKey := client.ObjectKeyFromObject(targetSync)

	nsFilter, err := newNamespaceFilter(targetSync.Spec.NamespaceExpression)
	if err != nil {
		return fmt.Errorf("failed to create namespace filter: %w", err)
	}

	sourceRestConfig, err := p.getSourceRestConfig(targetSync)
	if err != nil {
		return err
	}

	mgrOptions := manager.Options{
		LeaderElection:     false,
		MetricsBindAddress: "0",
	}
	mgr, err := ctrl.NewManager(sourceRestConfig, mgrOptions)
	if err != nil {
		return fmt.Errorf("failed to setup manager: %w", err)
	}

	log := logger.Reconciles("secret", "Secret")
	ctrl, err := NewSecretController(log, mgr.GetClient(), targetClient, mgr.GetScheme(), nsFilter, targetSync)
	if err != nil {
		return fmt.Errorf("failed to create secret controller: %w", err)
	}

	secrFilter, err := newSecretFilter(targetSync.Spec.SecretNameExpression)
	if err != nil {
		return fmt.Errorf("failed to create secret filter: %w", err)
	}

	err = builder.ControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(secrFilter)).
		WithLogConstructor(func(r *reconcile.Request) logr.Logger { return log.Logr() }).
		Complete(ctrl)
	if err != nil {
		return fmt.Errorf("failed to add secret controller to manager: %w", err)
	}

	controllerCtx, controllerCancelFunc := context.WithCancel(p.rootContext)
	item := secretControllerItem{
		cancel: controllerCancelFunc,
	}
	p.items[targetSyncKey] = &item

	go func() {
		defer p.deleteItem(targetSyncKey)

		logger.Info("Starting secret controller")
		err := mgr.Start(controllerCtx)
		if err != nil {
			logger.Error(err, "Secret controller stopped")
		}
	}()

	return nil
}

func (p *secretControllerPool) deleteSecretController(ctx context.Context, targetSyncKey client.ObjectKey) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{})

	item, ok := p.items[targetSyncKey]
	if !ok {
		logger.Info("Secret controller already gone")
		return nil
	}

	logger.Info("Stopping secret controller")
	item.cancel()

	p.deleteItem(targetSyncKey)
	return nil
}

func (p *secretControllerPool) getSourceRestConfig(targetSync *v1alpha1.TargetSync) (*rest.Config, error) {
	config := ctrl.GetConfigOrDie() // TODO get config to access the source cluster from TargetSync
	return config, nil
}
