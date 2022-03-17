package datasource

import (
	"context"
	"errors"
	"fmt"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/grafana/grafana/internal/components"
	"github.com/grafana/grafana/pkg/infra/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/schema"
	"github.com/grafana/grafana/pkg/services/sqlstore"
)

func init() {
	fmt.Println("regging datasource model")
	panic("regging datasource model")

	components.RegisterCoremodel(
		components.SchemaOpts{
			Type: schema.SchemaTypeThema,
			ThemaOpts: schema.ThemaLoaderOpts{
				SchemaFS:         cueFS,
				SchemaPath:       cuePath,
				SchemaVersion:    schemaVersion,
				GroupName:        groupName,
				GroupVersion:     groupVersion,
				SchemaOpenapi:    schemaOpenapi,
				SchemaType:       &DatasourceSpec{},
				SchemaObject:     &Datasource{},
				SchemaListObject: &DatasourceList{},
			},
		},
		func(s schema.ObjectSchema, l log.Logger) components.Coremodel {
			return NewCoremodel(s, l)
		},
	)
}

// Coremodel is the coremodel for Datasource component.
type Coremodel struct {
	schema schema.ObjectSchema
	store  components.Store
	client client.Client
	logger log.Logger
}

// NewCoremodel
func NewCoremodel(s schema.ObjectSchema, l log.Logger) *Coremodel {
	return &Coremodel{
		schema: s,
		logger: log.New("datasource"),
	}
}

// Schema returns the object schema for this Coremodel.
func (m *Coremodel) Schema() schema.ObjectSchema {
	return m.schema
}

// InjectStore
//
// TODO: currently this injects the sqlstore, but we need to inject a generic component store.
// We should do that with a storeset (a component similar to client set), in a separate PR.
func (m *Coremodel) InjectStore(store *sqlstore.SQLStore) error {
	m.store = NewStore(store, m.logger)
	return nil
}

// InjectClient is called when this coremodel is registered to controller manager.
// It will receive the client for the object kind controlled by coremodel, which can be used in reconciliation.
func (m *Coremodel) InjectClient(client client.Client) error {
	m.client = client
	return nil
}

// Reconcile implements Kubernetes controller reconciliation logic.
func (m *Coremodel) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	m.logger.Debug(
		"received reconciliation request",
		"request", req.String(),
	)

	var kubeVal Datasource
	if err := m.client.Get(ctx, req.NamespacedName, &kubeVal); kerrors.IsNotFound(err) {
		m.logger.Debug(
			"removing resource from local store",
			"request", req.String(),
		)

		// Since the object cannot be found in k8s anymore, it means it has been deleted.
		// We should reconcile this by deleting it from local storage as well.
		if err := m.store.Delete(ctx, req.NamespacedName); err != nil {
			m.logger.Error(
				"error removing resource from local store",
				"request", req.String(),
				"error", err,
			)

			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 1 * time.Minute,
			}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		m.logger.Error(
			"error fetching resource from kubernetes",
			"request", req.String(),
			"error", err,
		)

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 1 * time.Minute,
		}, err
	}

	var storeVal Datasource
	if err := m.store.Get(ctx, req.NamespacedName, &storeVal); errors.Is(err, models.ErrDataSourceNotFound) {
		m.logger.Debug(
			"inserting resource to local store",
			"request", req.String(),
		)

		// Since the object cannot be found in local storage, it means we need to create it.
		if err := m.store.Create(ctx, &kubeVal); err != nil {
			m.logger.Error(
				"error inserting resource to local store",
				"request", req.String(),
				"error", err,
			)

			return reconcile.Result{
				Requeue:      true,
				RequeueAfter: 1 * time.Minute,
			}, err
		}

		return reconcile.Result{}, nil
	} else if err != nil {
		m.logger.Error(
			"error fetching resource from local store",
			"request", req.String(),
			"error", err,
		)

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 1 * time.Minute,
		}, err
	}

	m.logger.Debug(
		"updating resource in local store",
		"request", req.String(),
	)

	// Make sure we merge values from both stores accordingly,
	// to account for any dynamic variables changing around.
	var resVal Datasource
	if err := mergeVals(storeVal, kubeVal, &resVal); err != nil {
		m.logger.Error(
			"error merging kubernetes and local values",
			"request", req.String(),
			"error", err,
		)

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 1 * time.Minute,
		}, err
	}

	if err := m.store.Update(ctx, &resVal); err != nil {
		m.logger.Error(
			"error updating resource in local store",
			"request", req.String(),
			"error", err,
		)

		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 1 * time.Minute,
		}, err
	}

	return reconcile.Result{}, nil
}

func mergeVals(store, kube Datasource, result *Datasource) error {
	// TODO: merge the values.
	*result = kube
	return nil
}
