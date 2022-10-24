// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"context"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func sync(ctx context.Context, secret *corev1.Secret, namespace *corev1.Namespace, targetClient client.Client) error {
	logger, ctx := logging.FromContextOrNew(ctx, []interface{}{})
	logger.Info("Syncing secret to namespace")

	// TODO common sync logic for secret controller and namespace controller, syncing a single secret namespace pair

	return nil
}
