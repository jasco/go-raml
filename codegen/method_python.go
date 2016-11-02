package codegen

import (
	"strings"

	"github.com/Jumpscale/go-raml/codegen/commons"
	"github.com/Jumpscale/go-raml/codegen/resource"
	"github.com/Jumpscale/go-raml/raml"
	log "github.com/Sirupsen/logrus"
)

// python server method
type pythonServerMethod struct {
	*resource.Method
	MiddlewaresArr []pythonMiddleware
}

// setup sets all needed variables
func (pm *pythonServerMethod) setup(apiDef *raml.APIDefinition, r *raml.Resource, rd *resource.Resource) error {
	// method name
	if len(pm.DisplayName) > 0 {
		pm.MethodName = displayNameToFuncName(pm.DisplayName)
	} else {
		pm.MethodName = snakeCaseResourceURI(r) + "_" + strings.ToLower(pm.Verb())
	}
	pm.Params = strings.Join(resource.GetResourceParams(r), ", ")
	pm.Endpoint = strings.Replace(pm.Endpoint, "{", "<", -1)
	pm.Endpoint = strings.Replace(pm.Endpoint, "}", ">", -1)

	// security middlewares
	for _, v := range pm.SecuredBy {
		if !validateSecurityScheme(v.Name, apiDef) {
			continue
		}
		// oauth2 middleware
		m, err := newPythonOauth2Middleware(v)
		if err != nil {
			log.Errorf("error creating middleware for method.err = %v", err)
			return err
		}
		pm.MiddlewaresArr = append(pm.MiddlewaresArr, m)
	}
	return nil
}

// defines a python client lib method
type pythonClientMethod struct {
	resource.Method
	PRArgs string // python requests's args
}

func newPythonClientMethod(r *raml.Resource, rd *resource.Resource, m *raml.Method, methodName, lang string) (resource.MethodInterface, error) {
	method := resource.NewMethod(r, rd, m, methodName, setBodyName)

	method.ResourcePath = commons.ParamizingURI(method.Endpoint, "+")

	name := commons.NormalizeURITitle(method.Endpoint)

	method.ReqBody = setBodyName(m.Bodies, name+methodName, "ReqBody")

	pcm := pythonClientMethod{Method: method}
	pcm.setup()
	return pcm, nil
}

func (pcm *pythonClientMethod) setup() {
	var prArgs string
	params := []string{"self"}

	// for method with request body, we add `data` argument
	if pcm.Verb() == "PUT" || pcm.Verb() == "POST" || pcm.Verb() == "PATCH" {
		params = append(params, "data")
		prArgs = ", data"
	}
	pcm.PRArgs = prArgs

	params = append(params, resource.GetResourceParams(pcm.Resource())...)
	pcm.Params = strings.Join(append(params, "headers=None, query_params=None"), ", ")

	if len(pcm.DisplayName) > 0 {
		pcm.MethodName = displayNameToFuncName(pcm.DisplayName)
	} else {
		pcm.MethodName = snakeCaseResourceURI(pcm.Resource()) + "_" + strings.ToLower(pcm.Verb())
	}
}

// create snake case function name from a resource URI
func snakeCaseResourceURI(r *raml.Resource) string {
	return _snakeCaseResourceURI(r, "")
}

func _snakeCaseResourceURI(r *raml.Resource, completeURI string) string {
	if r == nil {
		return completeURI
	}
	var snake string
	if len(r.URI) > 0 {
		uri := commons.NormalizeURI(r.URI)
		if r.Parent != nil { // not root resource, need to add "_"
			snake = "_"
		}

		if strings.HasPrefix(r.URI, "/{") {
			snake += "by" + strings.ToUpper(uri[:1])
		} else {
			snake += strings.ToLower(uri[:1])
		}

		if len(uri) > 1 { // append with the rest of uri
			snake += uri[1:]
		}
	}
	return _snakeCaseResourceURI(r.Parent, snake+completeURI)
}
