package service

import (
	"context"

	"github.com/cortezaproject/corteza-server/pkg/actionlog"
	a "github.com/cortezaproject/corteza-server/pkg/auth"
	"github.com/cortezaproject/corteza-server/pkg/dal"
	"github.com/cortezaproject/corteza-server/pkg/dal/capabilities"

	"github.com/cortezaproject/corteza-server/store"
	"github.com/cortezaproject/corteza-server/system/types"
)

type (
	dalConnection struct {
		actionlog actionlog.Recorder
		store     store.Storer
		ac        connectionAccessController
		dal       dalConnections

		primaryConnection types.DalConnection
	}

	connectionAccessController interface {
		CanGrant(context.Context) bool
		CanSearchDalConnections(ctx context.Context) bool

		CanCreateDalConnection(context.Context) bool
		CanReadDalConnection(context.Context, *types.DalConnection) bool
		CanUpdateDalConnection(context.Context, *types.DalConnection) bool
		CanDeleteDalConnection(context.Context, *types.DalConnection) bool
	}

	dalConnections interface {
		AddConnection(ctx context.Context, connectionID uint64, cp dal.ConnectionParams, dft dal.ConnectionDefaults, capabilities ...capabilities.Capability) (err error)
		UpdateConnection(ctx context.Context, connectionID uint64, cp dal.ConnectionParams, dft dal.ConnectionDefaults, capabilities ...capabilities.Capability) (err error)
		RemoveConnection(ctx context.Context, connectionID uint64) (err error)
	}
)

func Connection(ctx context.Context, pcOpts types.DalConnection, dal dalConnections) (*dalConnection, error) {
	out := &dalConnection{
		ac:                DefaultAccessControl,
		actionlog:         DefaultActionlog,
		store:             DefaultStore,
		dal:               dal,
		primaryConnection: pcOpts,
	}

	return out, out.reloadConnections(ctx)
}

func (svc *dalConnection) FindByID(ctx context.Context, ID uint64) (q *types.DalConnection, err error) {
	var (
		rProps = &dalConnectionActionProps{}
	)

	err = func() error {
		if ID == 0 {
			// primary; construct it since it doesn't actually exist
			aux := svc.primaryConnection
			q = &aux
			return nil
		}

		if ID == 0 {
			return DalConnectionErrInvalidID()
		}

		if q, err = store.LookupDalConnectionByID(ctx, svc.store, ID); err != nil {
			return DalConnectionErrInvalidID().Wrap(err)
		}

		rProps.setConnection(q)

		if !svc.ac.CanReadDalConnection(ctx, q) {
			return DalConnectionErrNotAllowedToRead(rProps)
		}

		return nil
	}()

	return q, svc.recordAction(ctx, rProps, DalConnectionActionLookup, err)
}

func (svc *dalConnection) FindPrimary(ctx context.Context) (q *types.DalConnection, err error) {
	var (
		rProps = &dalConnectionActionProps{}
	)

	err = func() error {
		// primary; construct it since it doesn't actually exist
		aux := svc.primaryConnection
		q = &aux
		return nil
	}()

	return q, svc.recordAction(ctx, rProps, DalConnectionActionLookup, err)
}

func (svc *dalConnection) Create(ctx context.Context, new *types.DalConnection) (q *types.DalConnection, err error) {
	var (
		qProps = &dalConnectionActionProps{new: new}
	)

	err = func() (err error) {
		if !svc.ac.CanCreateDalConnection(ctx) {
			return DalConnectionErrNotAllowedToCreate(qProps)
		}

		new.ID = nextID()
		new.CreatedAt = *now()
		new.CreatedBy = a.GetIdentityFromContext(ctx).Identity()

		if err = store.CreateDalConnection(ctx, svc.store, new); err != nil {
			return err
		}

		q = new

		return svc.dal.AddConnection(ctx, new.ID, new.Config.Connection, new.ConnectionDefaults(), new.ActiveCapabilities()...)
	}()

	return q, svc.recordAction(ctx, qProps, DalConnectionActionCreate, err)
}

func (svc *dalConnection) Update(ctx context.Context, upd *types.DalConnection) (q *types.DalConnection, err error) {
	var (
		qProps = &dalConnectionActionProps{update: upd}
		qq     *types.DalConnection
		e      error
	)

	err = func() (err error) {
		if qq, e = store.LookupDalConnectionByID(ctx, svc.store, upd.ID); e != nil {
			return DalConnectionErrNotFound(qProps)
		}

		if !svc.ac.CanUpdateDalConnection(ctx, qq) {
			return DalConnectionErrNotAllowedToUpdate(qProps)
		}

		upd.UpdatedAt = now()
		upd.CreatedAt = qq.CreatedAt
		upd.UpdatedBy = a.GetIdentityFromContext(ctx).Identity()

		if err = store.UpdateDalConnection(ctx, svc.store, upd); err != nil {
			return
		}

		q = upd

		return svc.dal.UpdateConnection(ctx, upd.ID, upd.Config.Connection, upd.ConnectionDefaults(), upd.ActiveCapabilities()...)
	}()

	return q, svc.recordAction(ctx, qProps, DalConnectionActionUpdate, err)
}

func (svc *dalConnection) DeleteByID(ctx context.Context, ID uint64) (err error) {
	var (
		qProps = &dalConnectionActionProps{}
		q      *types.DalConnection
	)

	err = func() (err error) {
		if ID == 0 {
			return DalConnectionErrInvalidID()
		}

		if q, err = store.LookupDalConnectionByID(ctx, svc.store, ID); err != nil {
			return
		}

		if !svc.ac.CanDeleteDalConnection(ctx, q) {
			return DalConnectionErrNotAllowedToDelete(qProps)
		}

		qProps.setConnection(q)

		q.DeletedAt = now()
		q.DeletedBy = a.GetIdentityFromContext(ctx).Identity()

		if err = store.UpdateDalConnection(ctx, svc.store, q); err != nil {
			return
		}

		return svc.dal.RemoveConnection(ctx, q.ID)
	}()

	return svc.recordAction(ctx, qProps, DalConnectionActionDelete, err)
}

func (svc *dalConnection) UndeleteByID(ctx context.Context, ID uint64) (err error) {
	var (
		qProps = &dalConnectionActionProps{}
		q      *types.DalConnection
	)

	err = func() (err error) {
		if ID == 0 {
			return DalConnectionErrInvalidID()
		}

		if q, err = store.LookupDalConnectionByID(ctx, svc.store, ID); err != nil {
			return
		}

		if !svc.ac.CanDeleteDalConnection(ctx, q) {
			return DalConnectionErrNotAllowedToUndelete(qProps)
		}

		qProps.setConnection(q)

		q.DeletedAt = nil
		q.UpdatedBy = a.GetIdentityFromContext(ctx).Identity()

		if err = store.UpdateDalConnection(ctx, svc.store, q); err != nil {
			return
		}

		return svc.dal.AddConnection(ctx, q.ID, q.Config.Connection, q.ConnectionDefaults(), q.ActiveCapabilities()...)
	}()

	return svc.recordAction(ctx, qProps, DalConnectionActionDelete, err)
}

func (svc *dalConnection) Search(ctx context.Context, filter types.DalConnectionFilter) (r types.DalConnectionSet, f types.DalConnectionFilter, err error) {
	var (
		aProps = &dalConnectionActionProps{search: &filter}
	)

	// For each fetched item, store backend will check if it is valid or not
	filter.Check = func(res *types.DalConnection) (bool, error) {
		if !svc.ac.CanReadDalConnection(ctx, res) {
			return false, nil
		}

		return true, nil
	}

	err = func() error {
		if !svc.ac.CanSearchDalConnections(ctx) {
			return DalConnectionErrNotAllowedToSearch()
		}

		if r, f, err = store.SearchDalConnections(ctx, svc.store, filter); err != nil {
			return err
		}

		return nil
	}()

	return r, f, svc.recordAction(ctx, aProps, DalConnectionActionSearch, err)
}

func (svc *dalConnection) reloadConnections(ctx context.Context) (err error) {
	// Get all available connections
	cc, _, err := store.SearchDalConnections(ctx, svc.store, types.DalConnectionFilter{})
	if err != nil {
		return
	}

	for _, c := range cc {
		if err = svc.dal.AddConnection(ctx, c.ID, c.Config.Connection, c.ConnectionDefaults(), c.ActiveCapabilities()...); err != nil {
			return
		}
	}

	return
}
