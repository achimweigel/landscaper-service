// SPDX-FileCopyrightText: 2022 "SAP SE or an SAP affiliate company and Gardener contributors"
//
// SPDX-License-Identifier: Apache-2.0

package targetsync

import (
	"regexp"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type namespaceFilter struct {
	namespaceExpression string
	compiledExpression  *regexp.Regexp
}

func newNamespaceFilter(namespaceExpression string) (*namespaceFilter, error) {
	compiledExpression, err := regexp.Compile(namespaceExpression)
	if err != nil {
		return nil, err
	}

	return &namespaceFilter{
		namespaceExpression: namespaceExpression,
		compiledExpression:  compiledExpression,
	}, nil
}

func (p *namespaceFilter) shouldBeProcessed(obj client.Object) bool {
	return p.compiledExpression.MatchString(obj.GetName())
}
