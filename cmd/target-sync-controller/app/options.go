// SPDX-FileCopyrightText: 2021 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package app

import (
	"context"
	goflag "flag"

	"github.com/gardener/landscaper/controller-utils/pkg/logging"

	flag "github.com/spf13/pflag"
	ctrl "sigs.k8s.io/controller-runtime"
)

// options holds the landscaper service controller options
type options struct {
	Log logging.Logger // Log is the logger instance
}

// NewOptions returns a new options instance
func NewOptions() *options {
	return &options{}
}

// AddFlags adds flags passed via command line
func (o *options) AddFlags(fs *flag.FlagSet) {
	logging.InitFlags(fs)
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
}

// Complete initializes the options instance and validates flags
func (o *options) Complete(ctx context.Context) error {
	log, err := logging.GetLogger()
	if err != nil {
		return err
	}
	o.Log = log
	ctrl.SetLogger(log.Logr())

	if err != nil {
		return err
	}

	err = o.validate()
	return err
}

func (o *options) validate() error {
	return nil
}
