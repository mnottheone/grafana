package components

import (
	"context"

	"github.com/grafana/grafana/pkg/schema"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Coremodel is an interface that must be implemented by each coremodel.
type Coremodel interface {
	// Schema should return coremodel's schema.
	Schema() schema.ObjectSchema
}

// SchemaLoader is a generic schema loader, that can load different schema types.
type SchemaLoader interface {
	LoadSchema(
		context.Context, schema.SchemaType, schema.ThemaLoaderOpts, schema.GoLoaderOpts,
	) (schema.ObjectSchema, error)
}

// Store is a generic durable storage for coremodels.
//
// TODO: I think we should define a generic store interface similar to k8s rest.Interface
// and have storeset around (similar to clientset) from which we can grab specific store implementation for schema.
type Store interface {
	// Get retrieves a coremodel with specified namespaced name from the store into the object.
	Get(context.Context, types.NamespacedName, runtime.Object) error

	// Delete deletes the coremodel with specified namespaced name from the store.
	Delete(context.Context, types.NamespacedName) error

	// Create creates a new coremodel object in the store.
	Create(context.Context, runtime.Object) error

	// Update updates the coremodel object in the store.
	Update(context.Context, runtime.Object) error
}
