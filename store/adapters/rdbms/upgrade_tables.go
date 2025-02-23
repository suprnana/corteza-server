package rdbms

import (
	. "github.com/cortezaproject/corteza-server/store/adapters/rdbms/ddl"
)

const (
	handleLength   = 64
	resourceLength = 512

	// https://www.rfc-editor.org/errata/eid1690
	emailLength = 254

	// Enough for IPv6, ports, delimiters, IPv4-mapped IPV6 addresses...
	ipAddrLength = 64

	// IETF language tag doesn't specify a hard limit (there can be a lot of modifiers)
	// so I can't put a strict limit.
	// Omiting the bits of the specs that don't have a limited length a size of 32 would suffice.
	// Going a bit extra for future proofing
	languageTagLength = 128

	// Some keys may introduce generated identifiers which may cause the size
	// to inflate.
	languageKeyLength = 256

	urlLength      = 2048
	locationLength = 256
)

// Tables fn holds a list of all tables that need to be created
func Tables() []*Table {
	return []*Table{
		tableUsers(),
		tableDalConnections(),
		tableDalSensitivityLevels(),
		tableCredentials(),
		tableAuthClients(),
		tableAuthConfirmedClients(),
		tableAuthSessions(),
		tableAuthOA2Tokens(),
		tableRoles(),
		tableRoleMembers(),
		tableApplications(),
		tableReminders(),
		tableAttachments(),
		tableActionLog(),
		tableRbacRules(),
		tableSettings(),
		tableLabels(),
		tableFlags(),
		tableTemplates(),
		tableReports(),
		tableResourceTranslations(),
		tableComposeAttachment(),
		tableComposeChart(),
		tableComposeModule(),
		tableComposeModuleField(),
		tableComposeNamespace(),
		tableComposePage(),
		tableComposeRecord(),
		tableComposeRecordValue(),
		tableFederationModuleShared(),
		tableFederationModuleExposed(),
		tableFederationModuleMapping(),
		tableFederationNodes(),
		tableFederationNodesSync(),
		tableAutomationWorkflows(),
		tableAutomationTriggers(),
		tableAutomationSessions(),
		//tableAutomationState(),
		tableMessagebusQueue(),
		tableMessagebusQueuemessage(),
		tableApigwRoute(),
		tableApigwFilter(),
		tableResourceActivityLog(),
	}
}

func tableUsers() *Table {
	return TableDef("users",
		ID,
		ColumnDef("email", ColumnTypeVarchar, ColumnTypeLength(emailLength)),
		ColumnDef("email_confirmed", ColumnTypeBoolean, DefaultValue("false")),
		ColumnDef("username", ColumnTypeText),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("kind", ColumnTypeVarchar, ColumnTypeLength(8)),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("suspended_at", ColumnTypeTimestamp, Null),
		CUDTimestamps,

		AddIndex("unique_email", IExpr("LOWER(email)"), IWhere("LENGTH(email) > 0 AND deleted_at IS NULL AND suspended_at IS NULL")),
		AddIndex("unique_username", IExpr("LOWER(username)"), IWhere("LENGTH(username) > 0 AND deleted_at IS NULL AND suspended_at IS NULL")),
		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL AND suspended_at IS NULL")),
	)
}

func tableDalConnections() *Table {
	return TableDef("dal_connections",
		ID,

		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("type", ColumnTypeText),

		ColumnDef("location", ColumnTypeJson),
		ColumnDef("ownership", ColumnTypeText),
		ColumnDef("sensitivity_level", ColumnTypeIdentifier),

		ColumnDef("config", ColumnTypeJson),
		ColumnDef("capabilities", ColumnTypeJson),
		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableDalSensitivityLevels() *Table {
	return TableDef("dal_sensitivity_levels",
		ID,

		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("level", ColumnTypeInteger),

		ColumnDef("meta", ColumnTypeJson),

		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableCredentials() *Table {
	return TableDef(`credentials`,
		ID,
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		ColumnDef("label", ColumnTypeText),
		ColumnDef("kind", ColumnTypeVarchar, ColumnTypeLength(128)),
		ColumnDef("credentials", ColumnTypeText),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("expires_at", ColumnTypeTimestamp, Null),
		ColumnDef("last_used_at", ColumnTypeTimestamp, Null),
		CUDTimestamps,

		AddIndex("owner_kind", IColumn("rel_owner", "kind"), IWhere("deleted_at IS NULL")),
	)
}

func tableAuthClients() *Table {
	return TableDef(`auth_clients`,
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("secret", ColumnTypeVarchar, ColumnTypeLength(64)),
		ColumnDef("scope", ColumnTypeVarchar, ColumnTypeLength(512)),
		ColumnDef("valid_grant", ColumnTypeVarchar, ColumnTypeLength(32)),
		ColumnDef("redirect_uri", ColumnTypeText),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("trusted", ColumnTypeBoolean),
		ColumnDef("valid_from", ColumnTypeTimestamp, Null),
		ColumnDef("expires_at", ColumnTypeTimestamp, Null),
		ColumnDef("security", ColumnTypeJson),
		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableAuthConfirmedClients() *Table {
	return TableDef(`auth_confirmed_clients`,
		ColumnDef("rel_user", ColumnTypeIdentifier),
		ColumnDef("rel_client", ColumnTypeIdentifier),
		ColumnDef("confirmed_at", ColumnTypeTimestamp),

		PrimaryKey(IColumn("rel_user", "rel_client")),
	)
}

func tableAuthSessions() *Table {
	return TableDef(`auth_sessions`,
		ColumnDef("id", ColumnTypeVarchar, ColumnTypeLength(64)),
		ColumnDef("data", ColumnTypeBinary),
		ColumnDef("rel_user", ColumnTypeIdentifier),
		ColumnDef("created_at", ColumnTypeTimestamp),
		ColumnDef("expires_at", ColumnTypeTimestamp),
		ColumnDef("remote_addr", ColumnTypeVarchar, ColumnTypeLength(ipAddrLength)),
		ColumnDef("user_agent", ColumnTypeText),

		AddIndex("expires_at", IColumn("expires_at")),
		AddIndex("user", IColumn("rel_user")),
		PrimaryKey(IColumn("id")),
	)
}

func tableAuthOA2Tokens() *Table {
	return TableDef(`auth_oa2tokens`,
		ID,

		ColumnDef("code", ColumnTypeVarchar, ColumnTypeLength(48)),
		ColumnDef("access", ColumnTypeVarchar, ColumnTypeLength(2048)),
		ColumnDef("refresh", ColumnTypeVarchar, ColumnTypeLength(48)),
		ColumnDef("data", ColumnTypeJson),
		ColumnDef("remote_addr", ColumnTypeVarchar, ColumnTypeLength(ipAddrLength)),
		ColumnDef("user_agent", ColumnTypeText),

		ColumnDef("rel_client", ColumnTypeIdentifier),
		ColumnDef("rel_user", ColumnTypeIdentifier),
		ColumnDef("created_at", ColumnTypeTimestamp),
		ColumnDef("expires_at", ColumnTypeTimestamp),

		AddIndex("expires_at", IColumn("expires_at")),
		AddIndex("code", IColumn("code")),
		AddIndex("access", IColumn("access")),
		AddIndex("refresh", IColumn("refresh")),
		AddIndex("client", IColumn("rel_client")),
		AddIndex("user", IColumn("rel_user")),
	)
}

func tableRoles() *Table {
	return TableDef(`roles`,
		ID,
		ColumnDef("name", ColumnTypeText),
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("archived_at", ColumnTypeTimestamp, Null),
		CUDTimestamps,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL AND archived_at IS NULL")),
	)
}

func tableRoleMembers() *Table {
	return TableDef(`role_members`,
		ColumnDef("rel_role", ColumnTypeIdentifier),
		ColumnDef("rel_user", ColumnTypeIdentifier),

		AddIndex("unique_membership", IColumn("rel_role", "rel_user")),
	)
}

func tableApplications() *Table {
	return TableDef("applications",
		ID,
		ColumnDef("name", ColumnTypeText),
		ColumnDef("enabled", ColumnTypeBoolean, DefaultValue("true")),
		ColumnDef("weight", ColumnTypeInteger, DefaultValue("0")),
		ColumnDef("unify", ColumnTypeJson),
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		CUDTimestamps,
	)
}

func tableReminders() *Table {
	return TableDef("reminders",
		ID,
		ColumnDef("resource", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("payload", ColumnTypeJson),
		ColumnDef("snooze_count", ColumnTypeInteger, DefaultValue("0")),
		ColumnDef("assigned_to", ColumnTypeIdentifier, DefaultValue("0")),
		ColumnDef("assigned_by", ColumnTypeIdentifier, DefaultValue("0")),
		ColumnDef("assigned_at", ColumnTypeTimestamp),
		ColumnDef("remind_at", ColumnTypeTimestamp, Null),
		ColumnDef("dismissed_at", ColumnTypeTimestamp, Null),
		ColumnDef("dismissed_by", ColumnTypeIdentifier, DefaultValue("0")),
		CUDTimestamps,

		AddIndex("resource", IColumn("resource")),
		AddIndex("assignee", IColumn("assigned_to")),
	)
}

func tableAttachments() *Table {
	return TableDef("attachments",
		ID,
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		ColumnDef("kind", ColumnTypeText),
		ColumnDef("url", ColumnTypeText),
		ColumnDef("preview_url", ColumnTypeText),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("meta", ColumnTypeJson),
		CUDTimestamps,
	)
}

func tableActionLog() *Table {
	return TableDef("actionlog",
		ID,
		ColumnDef("ts", ColumnTypeTimestamp),
		ColumnDef("actor_ip_addr", ColumnTypeVarchar, ColumnTypeLength(ipAddrLength)),
		ColumnDef("actor_id", ColumnTypeIdentifier),
		ColumnDef("request_origin", ColumnTypeVarchar, ColumnTypeLength(32)),
		ColumnDef("request_id", ColumnTypeVarchar, ColumnTypeLength(256)),
		ColumnDef("resource", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("action", ColumnTypeVarchar, ColumnTypeLength(64)),
		ColumnDef("error", ColumnTypeText),
		ColumnDef("severity", ColumnTypeInteger),
		ColumnDef("description", ColumnTypeText),
		ColumnDef("meta", ColumnTypeJson),

		AddIndex("ts", IColumn("ts")),
		AddIndex("request_origin", IColumn("request_origin")),
		AddIndex("actor_id", IColumn("actor_id")),
		AddIndex("resource", IColumn("resource")),
		AddIndex("action", IColumn("action")),
	)
}

func tableRbacRules() *Table {
	return TableDef("rbac_rules",
		ColumnDef("rel_role", ColumnTypeIdentifier),
		ColumnDef("resource", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("operation", ColumnTypeVarchar, ColumnTypeLength(50)),
		ColumnDef("access", ColumnTypeInteger),

		PrimaryKey(IColumn("rel_role", "resource", "operation")),
	)
}

func tableSettings() *Table {
	return TableDef("settings",
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("value", ColumnTypeJson),
		ColumnDef("updated_by", ColumnTypeIdentifier),
		ColumnDef("updated_at", ColumnTypeTimestamp),

		AddIndex("unique_name", IExpr("LOWER(name)"), IColumn("rel_owner")),
	)
}

func tableLabels() *Table {
	return TableDef("labels",
		ColumnDef("kind", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("rel_resource", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("value", ColumnTypeText),

		AddIndex("unique_kind_res_name", IColumn("kind", "rel_resource"), IExpr("LOWER(name)")),
	)
}

func tableFlags() *Table {
	return TableDef("flags",
		ColumnDef("kind", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("rel_resource", ColumnTypeIdentifier),
		ColumnDef("owned_by", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("active", ColumnTypeBoolean),

		AddIndex("unique_kind_res_owner_name", IColumn("kind", "rel_resource", "owned_by"), IExpr("LOWER(name)")),
	)
}

func tableTemplates() *Table {
	return TableDef("templates",
		ID,
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("language", ColumnTypeText),
		ColumnDef("type", ColumnTypeText),
		ColumnDef("partial", ColumnTypeBoolean),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("template", ColumnTypeText),
		CUDTimestamps,
		ColumnDef("last_used_at", ColumnTypeTimestamp, Null),

		AddIndex("unique_language_handle", IColumn("language"), IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableReports() *Table {
	return TableDef("reports",
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("scenarios", ColumnTypeJson),
		ColumnDef("sources", ColumnTypeJson),
		ColumnDef("blocks", ColumnTypeJson),

		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableResourceTranslations() *Table {
	return TableDef("resource_translations",
		ID,

		ColumnDef("lang", ColumnTypeVarchar, ColumnTypeLength(languageTagLength)),
		ColumnDef("resource", ColumnTypeVarchar, ColumnTypeLength(resourceLength)),
		ColumnDef("k", ColumnTypeVarchar, ColumnTypeLength(languageKeyLength)),
		ColumnDef("message", ColumnTypeText),

		CUDTimestamps,
		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDUsers,

		AddIndex("unique_translation", IExpr("LOWER(lang)"), IExpr("LOWER(resource)"), IExpr("LOWER(k)")),
	)
}

func tableComposeAttachment() *Table {
	// @todo merge with general attachment table

	return TableDef("compose_attachment",
		ID,
		ColumnDef("rel_namespace", ColumnTypeIdentifier),
		ColumnDef("rel_owner", ColumnTypeIdentifier),
		ColumnDef("kind", ColumnTypeText),
		ColumnDef("url", ColumnTypeText),
		ColumnDef("preview_url", ColumnTypeText),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("meta", ColumnTypeJson),
		CUDTimestamps,

		AddIndex("namespace", IColumn("rel_namespace")),
	)
}

func tableComposeChart() *Table {
	return TableDef("compose_chart",
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("rel_namespace", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("config", ColumnTypeJson),
		CUDTimestamps,

		AddIndex("namespace", IColumn("rel_namespace")),
		AddIndex("unique_handle", IColumn("rel_namespace"), IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableComposeModule() *Table {
	return TableDef("compose_module",
		ID,
		ColumnDef("rel_namespace", ColumnTypeIdentifier),
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("model_config", ColumnTypeJson),
		CUDTimestamps,

		AddIndex("namespace", IColumn("rel_namespace")),
		AddIndex("unique_handle", IColumn("rel_namespace"), IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableComposeModuleField() *Table {
	return TableDef("compose_module_field",
		ID,

		ColumnDef("rel_module", ColumnTypeIdentifier),
		ColumnDef("place", ColumnTypeInteger),
		ColumnDef("kind", ColumnTypeText),
		ColumnDef("options", ColumnTypeJson),
		ColumnDef("encoding_strategy", ColumnTypeJson),
		ColumnDef("default_value", ColumnTypeJson),
		ColumnDef("expressions", ColumnTypeJson),
		ColumnDef("name", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("label", ColumnTypeText),
		ColumnDef("is_private", ColumnTypeBoolean),
		ColumnDef("is_required", ColumnTypeBoolean),
		ColumnDef("is_visible", ColumnTypeBoolean),
		ColumnDef("is_multi", ColumnTypeBoolean),

		CUDTimestamps,

		AddIndex("unique_name", IColumn("rel_module"), IExpr("LOWER(name)"), IWhere("LENGTH(name) > 0 AND deleted_at IS NULL")),
		AddIndex("module", IColumn("rel_module")),
	)
}

func tableComposeNamespace() *Table {
	return TableDef("compose_namespace",
		ID,
		ColumnDef("name", ColumnTypeText),
		ColumnDef("slug", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("meta", ColumnTypeJson),
		CUDTimestamps,

		AddIndex("unique_slug", IExpr("LOWER(slug)"), IWhere("LENGTH(slug) > 0 AND deleted_at IS NULL")),
	)
}

func tableComposePage() *Table {
	return TableDef("compose_page",
		ID,
		ColumnDef("title", ColumnTypeText),
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("description", ColumnTypeText),
		ColumnDef("rel_namespace", ColumnTypeIdentifier),
		ColumnDef("rel_module", ColumnTypeIdentifier),
		ColumnDef("self_id", ColumnTypeIdentifier),
		ColumnDef("config", ColumnTypeJson),
		ColumnDef("blocks", ColumnTypeJson),
		ColumnDef("visible", ColumnTypeBoolean),
		ColumnDef("weight", ColumnTypeInteger),
		CUDTimestamps,

		AddIndex("self", IColumn("self_id")),
		AddIndex("namespace", IColumn("rel_namespace")),
		AddIndex("module", IColumn("rel_module")),
		AddIndex("unique_handle", IColumn("rel_namespace"), IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableComposeRecord() *Table {
	return TableDef("compose_record",
		ID,
		ColumnDef("rel_namespace", ColumnTypeIdentifier),
		ColumnDef("module_id", ColumnTypeIdentifier),
		ColumnDef("values", ColumnTypeJson),
		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,

		AddIndex("namespace", IColumn("rel_namespace")),
		AddIndex("module", IColumn("module_id")),
		AddIndex("owner", IColumn("owned_by")),
	)
}

func tableComposeRecordValue() *Table {
	return TableDef("compose_record_value",
		ColumnDef("record_id", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeVarchar, ColumnTypeLength(64)),
		ColumnDef("value", ColumnTypeText, ColumnTypeFlag("mysqlLongText", true)),
		ColumnDef("ref", ColumnTypeIdentifier),
		ColumnDef("place", ColumnTypeInteger),
		ColumnDef("deleted_at", ColumnTypeTimestamp, Null),

		PrimaryKey(IColumn("record_id", "name", "place")),
		AddIndex("ref", IColumn("ref"), IWhere("ref > 0")),
	)
}

func tableFederationModuleShared() *Table {
	return TableDef("federation_module_shared",
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("rel_node", ColumnTypeIdentifier),
		ColumnDef("xref_module", ColumnTypeIdentifier),
		ColumnDef("fields", ColumnTypeJson),
		CUDTimestamps,
		CUDUsers,
	)
}

func tableFederationModuleExposed() *Table {
	return TableDef("federation_module_exposed",
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("rel_node", ColumnTypeIdentifier),
		ColumnDef("rel_compose_module", ColumnTypeIdentifier),
		ColumnDef("rel_compose_namespace", ColumnTypeIdentifier),
		ColumnDef("fields", ColumnTypeJson),
		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_node_compose_module", IColumn("rel_node", "rel_compose_module", "rel_compose_namespace")),
	)
}

func tableFederationModuleMapping() *Table {
	return TableDef("federation_module_mapping",
		ColumnDef("rel_federation_module", ColumnTypeIdentifier),
		ColumnDef("rel_compose_module", ColumnTypeIdentifier),
		ColumnDef("rel_compose_namespace", ColumnTypeIdentifier),
		ColumnDef("field_mapping", ColumnTypeJson),

		AddIndex("unique_module_compose_module", IColumn("rel_federation_module", "rel_compose_module", "rel_compose_namespace")),
	)
}

func tableFederationNodes() *Table {
	return TableDef("federation_nodes",
		ID,
		ColumnDef("shared_node_id", ColumnTypeIdentifier),
		ColumnDef("name", ColumnTypeText),
		ColumnDef("base_url", ColumnTypeText),
		ColumnDef("status", ColumnTypeText),
		ColumnDef("contact", ColumnTypeVarchar, ColumnTypeLength(emailLength)),
		ColumnDef("pair_token", ColumnTypeText),
		ColumnDef("auth_token", ColumnTypeText),

		CUDTimestamps,
		CUDUsers,
	)
}

func tableFederationNodesSync() *Table {
	return TableDef("federation_nodes_sync",
		ColumnDef("rel_node", ColumnTypeIdentifier),
		ColumnDef("rel_module", ColumnTypeIdentifier),
		ColumnDef("sync_type", ColumnTypeText),
		ColumnDef("sync_status", ColumnTypeText),
		ColumnDef("time_action", ColumnTypeTimestamp),
	)
}

func tableAutomationWorkflows() *Table {
	return TableDef("automation_workflows",
		ID,
		ColumnDef("handle", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("trace", ColumnTypeBoolean),
		ColumnDef("keep_sessions", ColumnTypeInteger),
		ColumnDef("scope", ColumnTypeJson),
		ColumnDef("steps", ColumnTypeJson),
		ColumnDef("paths", ColumnTypeJson),
		ColumnDef("issues", ColumnTypeJson),
		ColumnDef("run_as", ColumnTypeIdentifier),
		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,

		AddIndex("unique_handle", IExpr("LOWER(handle)"), IWhere("LENGTH(handle) > 0 AND deleted_at IS NULL")),
	)
}

func tableAutomationSessions() *Table {
	return TableDef("automation_sessions",
		ID,
		ColumnDef("rel_workflow", ColumnTypeIdentifier),
		ColumnDef("status", ColumnTypeInteger),
		ColumnDef("event_type", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("resource_type", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("input", ColumnTypeJson),
		ColumnDef("output", ColumnTypeJson),
		ColumnDef("stacktrace", ColumnTypeJson),
		ColumnDef("created_by", ColumnTypeIdentifier),
		ColumnDef("created_at", ColumnTypeTimestamp),
		ColumnDef("purge_at", ColumnTypeTimestamp, Null),
		ColumnDef("suspended_at", ColumnTypeTimestamp, Null),
		ColumnDef("completed_at", ColumnTypeTimestamp, Null),
		ColumnDef("error", ColumnTypeText),

		AddIndex("workflow", IColumn("rel_workflow")),
		AddIndex("event_type", IFieldFull(&IField{Field: "event_type", Length: handleLength})),
		AddIndex("resource_type", IFieldFull(&IField{Field: "resource_type", Length: handleLength})),
		AddIndex("status", IColumn("status")),
		AddIndex("created_at", IColumn("created_at")),
		AddIndex("completed_at", IColumn("completed_at")),
		AddIndex("suspended_at", IColumn("suspended_at")),
	)
}

func tableAutomationTriggers() *Table {
	return TableDef("automation_triggers",
		ID,
		ColumnDef("rel_workflow", ColumnTypeIdentifier),
		ColumnDef("rel_step", ColumnTypeIdentifier),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("resource_type", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("event_type", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("constraints", ColumnTypeJson),
		ColumnDef("input", ColumnTypeJson),
		ColumnDef("owned_by", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,

		AddIndex("workflow", IColumn("rel_workflow")),
	)
}

func tableMessagebusQueuemessage() *Table {
	return TableDef("queue_messages",
		ID,
		ColumnDef("queue", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("payload", ColumnTypeBinary),
		ColumnDef("created", ColumnTypeTimestamp),
		ColumnDef("processed", ColumnTypeTimestamp, Null),

		// AddIndex("processed", IColumn("processed")),
	)
}

func tableMessagebusQueue() *Table {
	return TableDef("queue_settings",
		ID,
		ColumnDef("consumer", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("queue", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("meta", ColumnTypeJson),
		CUDTimestamps,
		CUDUsers,
	)
}

func tableApigwRoute() *Table {
	return TableDef("apigw_routes",
		ID,
		ColumnDef("endpoint", ColumnTypeText, ColumnTypeLength(resourceLength)),
		ColumnDef("method", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("meta", ColumnTypeJson),
		ColumnDef("rel_group", ColumnTypeIdentifier),
		CUDTimestamps,
		CUDUsers,
	)
}

func tableApigwFilter() *Table {
	return TableDef("apigw_filters",
		ID,
		ColumnDef("rel_route", ColumnTypeIdentifier),
		ColumnDef("weight", ColumnTypeInteger),
		ColumnDef("kind", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("ref", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("enabled", ColumnTypeBoolean),
		ColumnDef("params", ColumnTypeJson),
		CUDTimestamps,
		CUDUsers,
	)
}

func tableResourceActivityLog() *Table {
	return TableDef("resource_activity_log",
		ID,
		ColumnDef("rel_resource", ColumnTypeIdentifier),
		ColumnDef("resource_type", ColumnTypeText, ColumnTypeLength(handleLength)),
		ColumnDef("resource_action", ColumnTypeVarchar, ColumnTypeLength(handleLength)),
		ColumnDef("ts", ColumnTypeTimestamp),
		ColumnDef("meta", ColumnTypeJson),

		AddIndex("rel_resource", IColumn("rel_resource")),
		AddIndex("ts", IColumn("ts")),
	)
}
