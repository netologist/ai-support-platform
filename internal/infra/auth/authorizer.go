package auth

import (
	"context"
	"strings"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"

	"github.com/netologist/ai-support-platform/internal/domain/entity"
	"github.com/netologist/ai-support-platform/internal/domain/service"
)

const modelDefinition = `
[request_definition]
r = sub, obj, act

[policy_definition]
p = sub, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`

type Authorizer struct {
	enforcer *casbin.Enforcer
}

func NewAuthorizer() (Authorizer, error) {
	model, err := casbinmodel.NewModelFromString(modelDefinition)
	if err != nil {
		return Authorizer{}, err
	}

	enforcer, err := casbin.NewEnforcer(model)
	if err != nil {
		return Authorizer{}, err
	}

	policies := [][]string{
		{"admin", "tickets", "read"},
		{"admin", "tickets", "create"},
		{"admin", "tickets", "update"},
		{"admin", "documents", "read"},
		{"admin", "documents", "create"},
		{"agent", "tickets", "read"},
		{"agent", "tickets", "create"},
		{"agent", "tickets", "update"},
		{"agent", "documents", "read"},
		{"agent", "documents", "create"},
		{"viewer", "tickets", "read"},
		{"viewer", "documents", "read"},
	}

	for _, policy := range policies {
		_, err = enforcer.AddPolicy(policy)
		if err != nil {
			return Authorizer{}, err
		}
	}

	for _, role := range []string{"admin", "agent", "viewer"} {
		_, err = enforcer.AddGroupingPolicy(role, role)
		if err != nil {
			return Authorizer{}, err
		}
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
