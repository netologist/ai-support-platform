package auth

import (
	"context"
	_ "embed"
	"strings"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	stringadapter "github.com/casbin/casbin/v2/persist/string-adapter"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

//go:embed casbin/model.conf
var modelDefinition string

//go:embed casbin/policy.csv
var policyDefinition string

type Authorizer struct {
	enforcer *casbin.Enforcer
}

func NewAuthorizer() (Authorizer, error) {
	model, err := casbinmodel.NewModelFromString(modelDefinition)
	if err != nil {
		return Authorizer{}, err
	}

	adapter := stringadapter.NewAdapter(policyDefinition)
	enforcer, err := casbin.NewEnforcer(model, adapter)
	if err != nil {
		return Authorizer{}, err
	}

	if err = enforcer.LoadPolicy(); err != nil {
		return Authorizer{}, err
	}

	return Authorizer{enforcer: enforcer}, nil
}

func (authorizer Authorizer) Authorize(_ context.Context, principal entity.Principal, resource string, action string) error {
	allowed, err := authorizer.enforcer.Enforce(strings.ToLower(principal.Role), resource, action)
	if err != nil {
		return err
	}

	if !allowed {
		return service.ErrPermissionDenied
	}

	return nil
}
