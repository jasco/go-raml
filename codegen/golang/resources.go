package golang

import (
	"sort"
	"strings"

	"github.com/Jumpscale/go-raml/codegen/commons"
	"github.com/Jumpscale/go-raml/codegen/resource"
	"github.com/Jumpscale/go-raml/raml"
)

const (
	resourceIfTemplate  = "./templates/server_resources_interface.tmpl" // resource interface template
	resourceAPITemplate = "./templates/server_resources_api.tmpl"       // resource API template
)

type goResource struct {
	*resource.Resource
}

// generate interface file of a resource
func (gr *goResource) generateInterfaceFile(directory string) error {
	filename := directory + "/" + strings.ToLower(gr.Name) + "_if.go"
	return commons.GenerateFile(gr, resourceIfTemplate, "resource_if_template", filename, true)
}

// generate API file of a resource
func (gr *goResource) generateAPIFile(directory string) error {
	filename := directory + "/" + strings.ToLower(gr.Name) + "_api.go"
	return commons.GenerateFile(gr, resourceAPITemplate, "resource_api_template", filename, false)
}

// generate Go representation of server's resource.
// A resource have two kind of files:
// - interface file
//		contains API interface and routing code
//		always regenerated
// - API implementation
//		implementation of the API interface.
//		Don't generate if the file already exist
func (gr *goResource) generate(r *raml.Resource, URI, dir string) error {
	gr.GenerateMethods(r, "go", newServerMethod, newGoClientMethod)
	if err := gr.generateInterfaceFile(dir); err != nil {
		return err
	}
	return gr.generateAPIFile(dir)
}

// InterfaceImportPaths returns all packages imported by
// this resource interface file
func (gr goResource) InterfaceImportPaths() []string {
	ip := map[string]struct{}{
		"net/http":               struct{}{},
		"github.com/gorilla/mux": struct{}{},
	}

	for _, v := range gr.Methods {
		gm := v.(serverMethod)

		// if has middleware, we need to import middleware helper library
		if len(gm.Middlewares) > 0 {
			ip["github.com/justinas/alice"] = struct{}{}
		}
		for _, sb := range gm.SecuredBy {
			if lib := libImportPath(globRootImportPath, sb.Name); lib != "" {
				ip[lib] = struct{}{}
			}
		}
	}

	// return sorted array for predictable order
	// we need it for unit test to always return same order
	return sortImportPaths(ip)
}

// APIImportPaths returns all packages that need to be imported
// by the API implementation
func (gr goResource) APILibImportPaths() []string {
	ip := map[string]struct{}{
		"net/http": struct{}{},
	}

	// methods
	for _, v := range gr.Methods {
		gm := v.(serverMethod)
		if gm.RespBody != "" || gm.ReqBody != "" {
			ip["encoding/json"] = struct{}{}
		}
		for lib := range gm.libImported(globRootImportPath) {
			ip[lib] = struct{}{}
		}
	}

	// return sorted array for predictable order
	// we need it for unit test to always return same order
	return sortImportPaths(ip)
}

func sortImportPaths(ip map[string]struct{}) []string {
	libs := []string{}
	for k := range ip {
		libs = append(libs, k)
	}
	sort.Strings(libs)
	return libs
}
