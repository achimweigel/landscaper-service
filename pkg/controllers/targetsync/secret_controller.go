// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	lssv1alpha1 "github.com/gardener/landscaper-service/pkg/apis/core/v1alpha1"
)

// SecretController watches certain secrets on a source cluster and syncs them to a target cluster
type SecretController struct {
	log             logging.Logger
	sourceClient    client.Client // client to read the original secret from the source cluster
	targetClient    client.Client // client to write target and secret to the target cluster
	namespaceFilter *namespaceFilter
	targetSync      *lssv1alpha1.TargetSync
}

// NewSecretController returns a new SecretController
func NewSecretController(logger logging.Logger, sourceClient, targetClient client.Client, scheme *runtime.Scheme,
	namespaceFilter *namespaceFilter, targetSync *lssv1alpha1.TargetSync) (reconcile.Reconciler, error) {

	ctrl := &SecretController{
		log:             logger,
		sourceClient:    sourceClient,
		targetClient:    targetClient,
		namespaceFilter: namespaceFilter,
		targetSync:      targetSync,
	}

	return ctrl, nil
}

// Reconcile reconciles requests for Secrets
func (c *SecretController) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger, ctx := c.log.StartReconcileAndAddToContext(ctx, req,
		keyTargetSync, client.ObjectKeyFromObject(c.targetSync).String())

	secret := &corev1.Secret{}
	if err := c.sourceClient.Get(ctx, req.NamespacedName, secret); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info(err.Error())
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	namespaceList := &corev1.NamespaceList{}
	if err := c.targetClient.List(ctx, namespaceList); err != nil {
		return reconcile.Result{}, err
	}

	for i := range namespaceList.Items {
		namespace := &namespaceList.Items[i]

		if c.namespaceFilter.shouldBeProcessed(namespace) {
			_, nsCtx := logging.FromContextOrNew(ctx, []interface{}{}, keyNamespace, namespace.Name)

			if err := sync(nsCtx, secret, namespace, c.targetClient); err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	return reconcile.Result{}, nil
}
