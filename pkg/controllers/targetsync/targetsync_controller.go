// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"

	kutils "github.com/gardener/landscaper/controller-utils/pkg/kubernetes"
	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lssv1alpha1 "github.com/gardener/landscaper-service/pkg/apis/core/v1alpha1"
	lsserrors "github.com/gardener/landscaper-service/pkg/apis/errors"
)

// Controller is the TargetSync controller
type Controller struct {
	log    logging.Logger
	client client.Client
}

// NewController returns a new TargetSync controller
func NewController(logger logging.Logger, c client.Client, scheme *runtime.Scheme) (reconcile.Reconciler, error) {
	ctrl := &Controller{
		log:    logger,
		client: c,
	}

	return ctrl, nil
}

// Reconcile reconciles requests for TargetSyncs
func (c *Controller) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req)

	targetSync := &lssv1alpha1.TargetSync{}
	if err := c.client.Get(ctx, req.NamespacedName, targetSync); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	// set finalizer
	if targetSync.DeletionTimestamp.IsZero() && !kutils.HasFinalizer(targetSync, lssv1alpha1.LandscaperServiceFinalizer) {
		controllerutil.AddFinalizer(targetSync, lssv1alpha1.LandscaperServiceFinalizer)
		if err := c.client.Update(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if targetSync.DeletionTimestamp.IsZero() {
		if err := c.handleReconcile(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
	} else {
		if err := c.handleDelete(ctx, targetSync); err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (c *Controller) handleReconcile(ctx context.Context, targetSync *lssv1alpha1.TargetSync) error {
	secretCtlrPool := getSecretControllerPool()

	targetSyncKey := client.ObjectKeyFromObject(targetSync)

	if !secretCtlrPool.containsItem(targetSyncKey) {
		// create new secret controller
		if err := secretCtlrPool.createSecretController(ctx, c.client, targetSync); err != nil {
			return err
		}

	} else if targetSync.Generation != targetSync.Status.ObservedGeneration {
		// replace secret controller
		if err := secretCtlrPool.deleteSecretController(ctx, client.ObjectKeyFromObject(targetSync)); err != nil {
			return err
		}

		if err := secretCtlrPool.createSecretController(ctx, c.client, targetSync); err != nil {
			return err
		}
	} else {
		return nil
	}

	targetSync.Status.ObservedGeneration = targetSync.GetGeneration()
	if err := c.client.Status().Update(ctx, targetSync); err != nil {
		return err
	}

	return nil
}

func (c *Controller) handleDelete(ctx context.Context, targetSync *lssv1alpha1.TargetSync) error {
	secretCtlrPool := getSecretControllerPool()

	targetSyncKey := client.ObjectKeyFromObject(targetSync)

	if err := secretCtlrPool.deleteSecretController(ctx, targetSyncKey); err != nil {
		return err
	}

	controllerutil.RemoveFinalizer(targetSync, lssv1alpha1.LandscaperServiceFinalizer)
	if err := c.client.Update(ctx, targetSync); err != nil {
		return lsserrors.NewWrappedError(err, "handleDelete", "RemoveFinalizer", err.Error())
	}

	return nil
}
