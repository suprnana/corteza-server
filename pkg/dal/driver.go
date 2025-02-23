package dal

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/cortezaproject/corteza-server/pkg/dal/capabilities"
	"github.com/cortezaproject/corteza-server/pkg/expr"
	"github.com/cortezaproject/corteza-server/pkg/filter"
	"go.uber.org/zap"
)

type (
	PKValues map[string]any

	ConnectionParams struct {
		Type   string         `json:"type"`
		Params map[string]any `json:"params"`
	}

	Connection interface {
		// Meta

		// Models returns all the models the underlying connection already supports
		//
		// This is useful when adding support for new models since we can find out what
		// can work out of the box.
		Models(context.Context) (ModelSet, error)

		// Capabilities returns all of the capabilities the given store supports
		Capabilities() capabilities.Set

		// Can returns true if this store can handle the given capabilities
		Can(capabilities ...capabilities.Capability) bool

		// DML stuff

		// Create stores the given data into the underlying database
		Create(ctx context.Context, m *Model, rr ...ValueGetter) error

		// Update updates the given value in the underlying connection
		Update(ctx context.Context, m *Model, r ValueGetter) error

		// Lookup returns one bit of data
		Lookup(context.Context, *Model, ValueGetter, ValueSetter) error

		// Search returns an iterator which can be used to access all if the bits
		Search(context.Context, *Model, filter.Filter) (Iterator, error)

		// Delete deletes the given value
		Delete(ctx context.Context, m *Model, pkv ValueGetter) error

		// Truncate deletes all the data for the given model
		Truncate(ctx context.Context, m *Model) error

		// DDL stuff

		// // returns all attribute types that driver supports
		// AttributeTypes() []data.AttributeType

		// CreateModel adds support for the given models to the underlying database
		//
		// The operation returns an error if any of the models already exists.
		CreateModel(context.Context, *Model, ...*Model) error

		// DeleteModel removes support for the given model from the underlying database
		DeleteModel(context.Context, *Model, ...*Model) error

		// UpdateModel requests for metadata changes to the existing model
		//
		// Only metadata (such as idents) are affected; attributes can not be changed here
		UpdateModel(ctx context.Context, old *Model, new *Model) error

		// UpdateModelAttribute requests for the model attribute change
		//
		// Specific operations require data transformations (type change).
		// Some basic ops. should be implemented on DB driver level, but greater controll can be
		// achieved via the trans functions.
		UpdateModelAttribute(ctx context.Context, sch *Model, old Attribute, new Attribute, trans ...TransformationFunction) error
	}

	ConnectionCloser interface {
		// Close closes the store connection allowing the driver to perform potential
		// cleanup operations
		Close(ctx context.Context) error
	}

	TransformationFunction func(*Model, Attribute, expr.TypedValue) (expr.TypedValue, bool, error)

	// Iterator provides an interface for loading data from the underlying store
	Iterator interface {
		Next(ctx context.Context) bool
		Err() error
		Scan(ValueSetter) error
		Close() error

		BackCursor(ValueGetter) (*filter.PagingCursor, error)
		ForwardCursor(ValueGetter) (*filter.PagingCursor, error)

		// // -1 means unknown
		// Total() int
		// Cursor() any
		// // ... do we need anything else here?
	}

	// Store provides an interface which CRS uses to interact with the underlying database

	ValueGetter interface {
		CountValues() map[string]uint
		GetValue(string, uint) (any, error)
	}

	ValueSetter interface {
		SetValue(string, uint, any) error
	}

	ConnectorFn func(ctx context.Context, dsn string, cc ...capabilities.Capability) (Connection, error)

	DriverConnectionParam struct {
		Key        string `json:"key"`
		ValueType  string `json:"valueType"`
		MultiValue bool   `json:"multiValue"`
	}

	DriverConnectionConfig struct {
		Type   string                  `json:"type"`
		Params []DriverConnectionParam `json:"params"`
	}

	Driver struct {
		Type         string                 `json:"type"`
		Connection   DriverConnectionConfig `json:"connection"`
		Capabilities capabilities.Set       `json:"capabilities"`
	}
)

var (
	registeredConnectors = make(map[string]ConnectorFn)
	registeredDrivers    = make(map[string]Driver)
)

func (pkv PKValues) CountValues() map[string]uint {
	c := make(map[string]uint)
	for k := range pkv {
		c[k] = 1
	}

	return c
}

func (pkv PKValues) GetValue(key string, _ uint) (any, error) {
	if val, has := pkv[key]; has {
		return val, nil
	} else {
		return nil, fmt.Errorf("missing")
	}
}

// RegisterConnector registers a new connector for the given DSN schema
//
// In case of a duplicate schema the latter overwrites the prior
func RegisterConnector(fn ConnectorFn, tt ...string) {
	for _, t := range tt {
		registeredConnectors[t] = fn
	}
}

func RegisterDriver(d Driver) {
	registeredDrivers[d.Type] = d
}

// connect opens a new StoreConnection for the given CRS
func connect(ctx context.Context, log *zap.Logger, isDevelopment bool, cp ConnectionParams, capabilities ...capabilities.Capability) (Connection, error) {
	if cp.Type != "corteza::dal:connection:dsn" {
		return nil, fmt.Errorf("cannot open connection: only DSN connections supported")
	}
	dsn := cp.Params["dsn"].(string)

	if isDevelopment {
		if strings.Contains(dsn, "{version}") {
			log.Warn("You're using DB_DSN with {version}, It is still in EXPERIMENTAL phase")
			log.Warn("Should be used only for development mode")
			log.Warn("You may experience instability")
		}
		expr := regexp.MustCompile(`[.\-]+`)
		version := expr.ReplaceAllString(os.Getenv("BUILD_VERSION"), "_")
		dsn = strings.Replace(dsn, "{version}", version, 1)
	}

	var storeType = strings.SplitN(dsn, "://", 2)[0]
	if storeType == "" {
		// Backward compatibility
		storeType = "mysql"
	}

	if conn, ok := registeredConnectors[storeType]; ok {
		return conn(ctx, dsn, capabilities...)
	} else {
		return nil, fmt.Errorf("unknown store type used: %q (check your storage configuration)", storeType)
	}
}

func NewDSNDriverConnectionConfig() DriverConnectionConfig {
	return DriverConnectionConfig{
		Type: "corteza::dal:connection:dsn",
		Params: []DriverConnectionParam{{
			Key:       "dsn",
			ValueType: "string",
		}},
	}
}
func NewHTTPDriverConnectionConfig() DriverConnectionConfig {
	panic("not implemented NewHTTPDriverConnectionConfig")
	return DriverConnectionConfig{
		Type:   "corteza::dal:connection:http",
		Params: []DriverConnectionParam{{}},
	}
}
func NewFederatedNodeDriverConnectionConfig() DriverConnectionConfig {
	panic("not implemented NewFederatedNodeDriverConnectionConfig")
	return DriverConnectionConfig{
		Type:   "corteza::dal:connection:federated-node",
		Params: []DriverConnectionParam{{}},
	}
}

func NewDSNConnection(dsn string) ConnectionParams {
	return ConnectionParams{
		Type: "corteza::dal:connection:dsn",
		Params: map[string]any{
			"dsn": dsn,
		},
	}
}

func NewHTTPConnection(url string, headers, query map[string][]string) ConnectionParams {
	return ConnectionParams{
		Type: "corteza::dal:connection:http",
		Params: map[string]any{
			"url":     url,
			"headers": headers,
			"query":   query,
		},
	}
}

func NewFederatedNodeCOnnection(url string, pairToken, authToken string) ConnectionParams {
	return ConnectionParams{
		Type: "corteza::dal:connection:federation-node",
		Params: map[string]any{
			"baseURL":   url,
			"pairToken": pairToken,
			"authToken": authToken,
		},
	}
}
