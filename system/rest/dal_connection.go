package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	federationService "github.com/cortezaproject/corteza-server/federation/service"
	federationTypes "github.com/cortezaproject/corteza-server/federation/types"
	"github.com/cortezaproject/corteza-server/pkg/api"
	"github.com/cortezaproject/corteza-server/pkg/dal"
	"github.com/cortezaproject/corteza-server/pkg/filter"
	"github.com/cortezaproject/corteza-server/pkg/handle"
	"github.com/cortezaproject/corteza-server/pkg/payload"
	"github.com/cortezaproject/corteza-server/system/rest/request"
	"github.com/cortezaproject/corteza-server/system/service"
	"github.com/cortezaproject/corteza-server/system/types"
	"github.com/modern-go/reflect2"
	"github.com/pkg/errors"
)

var _ = errors.Wrap

type (
	DalConnection struct {
		svc           connectionService
		federationSvc federationNodeService

		connectionAc connectionAccessController
	}

	connectionWrap struct {
		types.DalConnection

		Type    string `json:"type"`
		Primary bool   `json:"primary"`
	}
	connectionWrapSet []connectionWrap

	connectionSetPayload struct {
		Filter types.DalConnectionFilter `json:"filter"`
		Set    types.DalConnectionSet    `json:"set"`
	}

	connectionAccessController interface {
		CanCreateDalConnection(context.Context) bool
		CanUpdateDalConnection(context.Context, *types.DalConnection) bool
	}

	connectionService interface {
		FindByID(ctx context.Context, ID uint64) (*types.DalConnection, error)
		FindPrimary(ctx context.Context) (*types.DalConnection, error)
		Create(ctx context.Context, new *types.DalConnection) (*types.DalConnection, error)
		Update(ctx context.Context, upd *types.DalConnection) (*types.DalConnection, error)
		DeleteByID(ctx context.Context, ID uint64) error
		UndeleteByID(ctx context.Context, ID uint64) error
		Search(ctx context.Context, filter types.DalConnectionFilter) (types.DalConnectionSet, types.DalConnectionFilter, error)
	}

	federationNodeService interface {
		Search(ctx context.Context, filter federationTypes.NodeFilter) (set federationTypes.NodeSet, f federationTypes.NodeFilter, err error)
	}
)

func (DalConnection) New() *DalConnection {
	return &DalConnection{
		svc:           service.DefaultDalConnection,
		federationSvc: federationService.DefaultNode,

		connectionAc: service.DefaultAccessControl,
	}
}

func (ctrl DalConnection) List(ctx context.Context, r *request.DalConnectionList) (interface{}, error) {
	var (
		err            error
		dalConnections types.DalConnectionSet

		f = types.DalConnectionFilter{
			ConnectionID: payload.ParseUint64s(r.ConnectionID),
			Handle:       r.Handle,
			Type:         r.Type,

			Deleted: filter.State(r.Deleted),
		}
	)

	if f.Deleted == 0 {
		f.Deleted = filter.StateInclusive
	}

	dalConnections, err = ctrl.collectConnections(ctx, f)
	if err != nil {
		return nil, err
	}

	return ctrl.makeFilterPayload(ctx, dalConnections, f)
}

func (ctrl DalConnection) Create(ctx context.Context, r *request.DalConnectionCreate) (interface{}, error) {
	connection := &types.DalConnection{
		Name:   r.Name,
		Handle: r.Handle,
		Type:   r.Type,

		Location:         r.Location,
		Ownership:        r.Ownership,
		SensitivityLevel: r.SensitivityLevel,

		Config:       r.Config,
		Capabilities: r.Capabilities,
	}

	switch connection.Type {
	case types.DalConnectionResourceType:
		return ctrl.svc.Create(ctx, connection)
	default:
		return nil, fmt.Errorf("cannot create connection: unsupported connection type %s", connection.Type)
	}

}

func (ctrl DalConnection) Update(ctx context.Context, r *request.DalConnectionUpdate) (interface{}, error) {
	connection := &types.DalConnection{
		ID:     r.ConnectionID,
		Name:   r.Name,
		Handle: r.Handle,
		Type:   r.Type,

		Location:         r.Location,
		Ownership:        r.Ownership,
		SensitivityLevel: r.SensitivityLevel,

		Config:       r.Config,
		Capabilities: r.Capabilities,
	}

	return ctrl.svc.Update(ctx, connection)
}

func (ctrl DalConnection) Read(ctx context.Context, r *request.DalConnectionRead) (interface{}, error) {
	return ctrl.svc.FindByID(ctx, r.ConnectionID)
}

func (ctrl DalConnection) Delete(ctx context.Context, r *request.DalConnectionDelete) (interface{}, error) {
	return api.OK(), ctrl.svc.DeleteByID(ctx, r.ConnectionID)
}

func (ctrl DalConnection) Undelete(ctx context.Context, r *request.DalConnectionUndelete) (interface{}, error) {
	return api.OK(), ctrl.svc.UndeleteByID(ctx, r.ConnectionID)
}

func (ctrl DalConnection) makeFilterPayload(ctx context.Context, connections types.DalConnectionSet, f types.DalConnectionFilter) (out *connectionSetPayload, err error) {
	out = &connectionSetPayload{
		Set:    connections,
		Filter: f,
	}

	return
}

func (ctrl DalConnection) federatedNodeToConnection(f *federationTypes.Node) *types.DalConnection {
	h, _ := handle.Cast(nil, f.Name)

	return &types.DalConnection{
		ID:        f.ID,
		Name:      f.Name,
		Handle:    h,
		Type:      federationTypes.NodeResourceType,
		Location:  nil,
		Ownership: f.Contact,

		Config: types.ConnectionConfig{
			Connection: dal.NewFederatedNodeCOnnection(f.BaseURL, f.PairToken, f.AuthToken),
		},

		CreatedAt: f.CreatedAt,
		CreatedBy: f.CreatedBy,
		UpdatedAt: f.UpdatedAt,
		UpdatedBy: f.UpdatedBy,
		DeletedAt: f.DeletedAt,
		DeletedBy: f.DeletedBy,
	}
}

func (ctrl DalConnection) collectConnections(ctx context.Context, f types.DalConnectionFilter) (out types.DalConnectionSet, err error) {
	var (
		dalConnections       types.DalConnectionSet
		primaryDalConnection *types.DalConnection
		federatedNodes       federationTypes.NodeSet
	)

	if dalConnections, f, err = ctrl.svc.Search(ctx, f); err != nil {
		return nil, err
	}

	if primaryDalConnection, err = ctrl.svc.FindPrimary(ctx); err != nil {
		return nil, err
	}

	if !reflect2.IsNil(ctrl.federationSvc) {
		if federatedNodes, _, err = ctrl.federationSvc.Search(ctx, federationTypes.NodeFilter{
			// @todo IDs?
			Deleted: f.Deleted,
		}); err != nil {
			return nil, err
		}
	}

	out = append(out, primaryDalConnection)
	out = append(out, dalConnections...)

	// We're converting federation nodes to DAL connection structs so that we have
	// a unified output.
	//
	// Eventually federation nodes will become connections, so this is ok
	for _, f := range federatedNodes {
		out = append(out, ctrl.federatedNodeToConnection(f))
	}

	out = ctrl.filterConnections(out, f)

	return
}

func (ctrl DalConnection) filterConnections(baseConnections types.DalConnectionSet, f types.DalConnectionFilter) (out types.DalConnectionSet) {
	for _, conn := range baseConnections {
		include := true

		if len(f.ConnectionID) > 0 {
			include = include && ctrl.inIDSet(f.ConnectionID, conn.ID)
		}

		if f.Handle != "" {
			include = include && f.Handle == conn.Handle
		}

		if f.Type != "" {
			include = include && f.Type == conn.Type
		}

		{
			if f.Deleted == filter.StateExcluded {
				include = include && conn.DeletedAt == nil
			}

			if f.Deleted == filter.StateExclusive {
				include = include && conn.DeletedAt != nil
			}
		}

		if f.Check != nil {
			// @todo error handling here?
			tmp, _ := f.Check(conn)
			include = include && tmp
		}

		if include {
			out = append(out, conn)
		}
	}

	return
}

func (ctrl DalConnection) inIDSet(set []uint64, target uint64) (out bool) {
	for _, id := range set {
		out = out || id == target
	}

	return
}

func (ctrl DalConnection) serve(ctx context.Context, fn string, archive io.ReadSeeker, err error) (interface{}, error) {
	if err != nil {
		return nil, err
	}

	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Disposition", "attachment; filename="+fn)

		http.ServeContent(w, req, fn, time.Now(), archive)
	}, nil
}
