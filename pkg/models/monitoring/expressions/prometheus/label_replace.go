package prometheus

import (
	"fmt"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/storage/metric"
	"kubesphere.io/kubesphere/pkg/models/monitoring/expressions"
)

func init() {
	expressions.Register("prometheus", labelReplace)
}

func labelReplace(input, ns string) (string, error) {
	root, err := promql.ParseExpr(input)
	if err != nil {
		return "", err
	}

	SetRecursive(root, ns)
	if err != nil {
		return "", err
	}

	return root.String(), nil
}

// Inspired by https://github.com/openshift/prom-label-proxy
func SetRecursive(node promql.Node, namespace string) (err error) {
	switch n := node.(type) {
	case *promql.EvalStmt:
		if err := SetRecursive(n.Expr, namespace); err != nil {
			return err
		}
	case promql.Expressions:
		for _, e := range n {
			if err := SetRecursive(e, namespace); err != nil {
				return err
			}
		}
	case *promql.AggregateExpr:
		if err := SetRecursive(n.Expr, namespace); err != nil {
			return err
		}
	case *promql.BinaryExpr:
		if err := SetRecursive(n.LHS, namespace); err != nil {
			return err
		}
		if err := SetRecursive(n.RHS, namespace); err != nil {
			return err
		}
	case *promql.Call:
		if err := SetRecursive(n.Args, namespace); err != nil {
			return err
		}
	case *promql.ParenExpr:
		if err := SetRecursive(n.Expr, namespace); err != nil {
			return err
		}
	case *promql.UnaryExpr:
		if err := SetRecursive(n.Expr, namespace); err != nil {
			return err
		}
	case *promql.NumberLiteral, *promql.StringLiteral:
		// nothing to do
	case *promql.MatrixSelector:
		n.LabelMatchers = enforceLabelMatchers(n.LabelMatchers, namespace)
	case *promql.VectorSelector:
		n.LabelMatchers = enforceLabelMatchers(n.LabelMatchers, namespace)
	default:
		return fmt.Errorf("promql.Walk: unhandled node type %T", node)
	}
	return err
}

func enforceLabelMatchers(matchers metric.LabelMatchers, namespace string) metric.LabelMatchers {
	var found bool
	for i, m := range matchers {
		if m.Name == "namespace" {
			matchers[i] = &metric.LabelMatcher{
				Name:  "namespace",
				Type:  metric.Equal,
				Value: model.LabelValue(namespace),
			}
			found = true
			break
		}
	}

	if !found {
		matchers = append(matchers, &metric.LabelMatcher{
			Name:  "namespace",
			Type:  metric.Equal,
			Value: model.LabelValue(namespace),
		})
	}
	return matchers
}